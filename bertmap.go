package main

import (
	"fmt"
	"math/big"

	bert "github.com/kbudde/gobert"
	log "github.com/sirupsen/logrus"
)

// rabbitBERTReply (along with its RabbitReply interface
// implementation) allow parsing of BERT-encoded RabbitMQ replies in a
// way that's fully compatible with JSON parser from jsonmap.go
type rabbitBERTReply struct {
	body    []byte
	objects bert.Term
}

func makeBERTReply(body []byte) (RabbitReply, error) {
	rawObjects, err := bert.Decode(body)
	return &rabbitBERTReply{body, rawObjects}, err
}

func (rep *rabbitBERTReply) MakeStatsInfo(labels []string) []StatsInfo {
	rawObjects := rep.objects

	objects, ok := rawObjects.([]bert.Term)
	if !ok {
		log.WithField("got", rawObjects).Error("Statistics reply should contain a slice of objects")
		return make([]StatsInfo, 0)
	}

	statistics := make([]StatsInfo, 0, len(objects))

	for _, v := range objects {
		obj, ok := parseSingleStatsObject(v, labels)
		if !ok {
			log.WithField("got", v).Error("Ignoring unparseable stats object")
			continue
		}
		statistics = append(statistics, *obj)
	}

	return statistics
}

func (rep *rabbitBERTReply) MakeMap() MetricMap {
	flMap := make(MetricMap)
	term := rep.objects

	err := parseProplist(&flMap, "", term)
	log.WithField("error", err).Warn("Error parsing rabbitmq reply (bert, MakeMap)")
	return flMap
}

// iterateBertKV helps to traverse any map-like structures returned by
// RabbitMQ with a user-provided function. We need it because
// different versions of RabbitMQ can encode a map in a bunch of
// different ways:
// - proplist
// - proplist additionally wrapped in a {struct, ...} tuple
// - map type available in modern erlang versions
//
// Non-nil error return means that an object can't be interpreted as a map in any way
//
// Provided function can return 'false' value to stop traversal earlier
func iterateBertKV(obj interface{}, elemFunc func(string, interface{}) bool) error {
	switch obj := obj.(type) {
	case []bert.Term:
		pairs, ok := assertBertProplistPairs(obj)
		if !ok {
			return bertError("Doesn't look like a proplist", obj)
		}
		for _, v := range pairs {
			key, value, ok := assertBertKeyedTuple(v)
			if ok {
				needToContinue := elemFunc(key, value)
				if !needToContinue {
					return nil
				}
			}
		}
		return nil
	case bert.Map:
		for keyRaw, value := range obj {
			key, ok := parseBertStringy(keyRaw)
			if ok {
				needToContinue := elemFunc(key, value)
				if !needToContinue {
					return nil
				}
			}
		}
		return nil
	default:
		return bertError("Can't iterate over non-KV object", obj)
	}
}

// parseSingleStatsObject extracts information about a named RabbitMQ
// object: both its vhost/name information and then the usual
// MetricMap.
func parseSingleStatsObject(obj interface{}, labels []string) (*StatsInfo, bool) {
	var result StatsInfo
	var objectOk = true
	result.metrics = make(MetricMap)
	result.labels = make(map[string]string)
	for _, label := range labels {
		result.labels[label] = ""
	}
	err := iterateBertKV(obj, func(key string, value interface{}) bool {
		//Check if current key should be saved as label
		for _, label := range labels {
			if key == label {
				tmp, ok := parseBertStringy(value)
				if !ok {
					log.WithField("got", value).WithField("label", label).Error("Non-string field")
					objectOk = false
					return false
				}
				result.labels[label] = tmp
			}
		}

		arr, isSlice := assertBertSlice(value)
		_, iSPropList := assertBertProplistPairs(value)

		//save metrics for array length.
		// An array is a slice which is not a proplist.
		// Arrays with len()==0 are special. IsProplist is true
		if isSlice && (!iSPropList || len(arr) == 0) {
			result.metrics[key+"_len"] = float64(len(arr))
		}

		if floatValue, ok := parseFloaty(value); ok {
			result.metrics[key] = floatValue
			return true
		}

		// Nested structures don't need special
		// processing, so we fallback to generic
		// parser.
		if err := parseProplist(&result.metrics, key, value); err == nil {
			return true
		}
		return true
	})
	if err == nil && objectOk {
		return &result, true
	}
	return nil, false
}

// parseProplist descends into an erlang data structure and stores
// everything remotely resembling a float in a toMap.
func parseProplist(toMap *MetricMap, basename string, maybeProplist interface{}) error {
	prefix := ""
	if basename != "" {
		prefix = basename + "."
	}
	return iterateBertKV(maybeProplist, func(key string, value interface{}) bool {
		if floatValue, ok := parseFloaty(value); ok {
			(*toMap)[prefix+key] = floatValue
			return true
		}
		if arraySize, ok := parseArray(value); ok {
			(*toMap)[prefix+key+"_len"] = arraySize
		}

		err := parseProplist(toMap, prefix+key, value) // This can fail, but we don't care
		log.WithField("error", err).Debug("Error parsing rabbitmq reply (bert, parseProplist)")
		return true
	})
}

// assertBertSlice checks whether the provided value is something
// that's represented as a slice by BERT parcer (list or tuple).
func assertBertSlice(maybeSlice interface{}) ([]bert.Term, bool) {
	switch it := maybeSlice.(type) {
	case []bert.Term:
		return it, true
	default:
		return nil, false
	}
}

// assertBertKeyedTuple checks whether the provided value looks like
// an element of proplist - 2-element tuple where the first elemen is
// an atom.
func assertBertKeyedTuple(maybeTuple interface{}) (string, bert.Term, bool) {
	tuple, ok := assertBertSlice(maybeTuple)
	if !ok {
		return "", nil, false
	}
	if len(tuple) != 2 {
		return "", nil, false
	}
	key, ok := assertBertAtom(tuple[0])
	if !ok {
		return "", nil, false
	}
	return key, tuple[1], true
}

func assertBertAtom(val interface{}) (string, bool) {
	if atom, ok := val.(bert.Atom); ok {
		return string(atom), true
	}
	return "", false
}

// assertBertProplistPairs checks whether the provided value points to
// a proplist. Additional level of {struct, ...} wrapping can be
// removed in process.
func assertBertProplistPairs(maybeTaggedProplist interface{}) ([]bert.Term, bool) {
	terms, ok := assertBertSlice(maybeTaggedProplist)
	if !ok {
		return nil, false
	}

	if len(terms) == 0 {
		return terms, true
	}

	// Strip {struct, ...} tagging than is used to help RabbitMQ
	// JSON encoder
	key, value, ok := assertBertKeyedTuple(terms)
	if ok && key == "struct" {
		return assertBertProplistPairs(value)
	}

	// Minimal safety check - at least the first element should be
	// a proplist pair
	_, _, ok = assertBertKeyedTuple(terms[0])
	if ok {
		return terms, true
	}
	return nil, false
}

// parseArray tries to interpret the provided BERT value as an array.
// It returns the size of the array
func parseArray(arr interface{}) (float64, bool) {
	switch t := arr.(type) {
	case []bert.Term:
		_, isPropList := assertBertProplistPairs(t)
		if !isPropList || len(t) == 0 {
			return float64(len(t)), true
		}
	}
	return 0, false
}

// parseFloaty tries to interpret the provided BERT value as a
// float. Floats itself, integers and booleans are handled.
func parseFloaty(num interface{}) (float64, bool) {
	switch num := num.(type) {
	case int:
		return float64(num), true
	case int8:
		return float64(num), true
	case int16:
		return float64(num), true
	case int32:
		return float64(num), true
	case int64:
		return float64(num), true
	case uint:
		return float64(num), true
	case uint8:
		return float64(num), true
	case uint16:
		return float64(num), true
	case uint32:
		return float64(num), true
	case uint64:
		return float64(num), true
	case float32:
		return float64(num), true
	case float64:
		return num, true
	case bert.Atom:
		if num == bert.TrueAtom {
			return 1, true
		} else if num == bert.FalseAtom {
			return 0, true
		}
	case big.Int:
		bigFloat := new(big.Float).SetInt(&num)
		result, _ := bigFloat.Float64()
		return result, true
	}
	return 0, false
}

// parseBertStringy tries to extract an Erlang value that can be
// represented as a Go string (binary or atom).
func parseBertStringy(val interface{}) (string, bool) {
	if stringer, ok := val.(fmt.Stringer); ok {
		return stringer.String(), true
	} else if atom, ok := val.(bert.Atom); ok {
		return string(atom), true
	} else if s, ok := val.(string); ok {
		return s, true
	}
	return "", false
}

type bertDecodeError struct {
	message string
	object  interface{}
}

func (err *bertDecodeError) Error() string {
	return fmt.Sprintf("%s while decoding: %s", err.message, err.object)
}

func bertError(message string, object interface{}) error {
	return &bertDecodeError{message, object}
}

func (rep *rabbitBERTReply) GetString(label string) (string, bool) {
	var resValue string
	var result bool
	result = false

	err := iterateBertKV(rep.objects, func(key string, value interface{}) bool {
		//Check if current key should be saved as label

		if key == label {
			tmp, ok := parseBertStringy(value)
			if !ok {
				return false
			}
			resValue = tmp
			result = true
			return false
		}
		return true
	})
	log.WithField("error", err).Warn("Error parsing rabbitmq reply (bert, GetString)")
	return resValue, result
}
