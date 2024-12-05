package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"log/slog"
)

type rabbitJSONReply struct {
	body []byte
	keys map[string]interface{}
}

func makeJSONReply(body []byte) (RabbitReply, error) {
	return &rabbitJSONReply{body, nil}, nil
}

//MakeStatsInfo creates a slice of StatsInfo from json input. Only keys with float values are mapped into `metrics`.
func (rep *rabbitJSONReply) MakeStatsInfo(labels []string) []StatsInfo {
	var statistics []StatsInfo
	var jsonArr []map[string]interface{}
	decoder := json.NewDecoder(bytes.NewBuffer(rep.body))
	if decoder == nil {
		slog.Error("JSON decoder not iniatilized")
		return make([]StatsInfo, 0)
	}

	if err := decoder.Decode(&jsonArr); err != nil {
		slog.Error("Error while decoding json")
		slog.Any("error", err)
		return make([]StatsInfo, 0)
	}

	for _, el := range jsonArr {
		field := ""
		if _, fieldName := el["name"]; fieldName {
			field = "name"
		}
		if _, fieldID := el["id"]; fieldID {
			field = "id"
		}
		if field != "" {
			slog.Debug("Iterate over array", "element", el, "vhost", el["vhost"], field, el[field])
			statsinfo := StatsInfo{}
			statsinfo.labels = make(map[string]string)

			for _, label := range labels {
				statsinfo.labels[label] = ""
				if tmp, ok := el[label]; ok {
					if v, ok := tmp.(string); ok {
						statsinfo.labels[label] = v
					} else if v, ok := tmp.(bool); ok {
						statsinfo.labels[label] = strconv.FormatBool(v)
					}
				}
			}

			statsinfo.metrics = make(MetricMap)
			addFields(&statsinfo.metrics, "", el)
			statistics = append(statistics, statsinfo)
		}
	}

	return statistics
}

//MakeMap creates a map from json input. Only keys with float values are mapped.
func (rep *rabbitJSONReply) MakeMap() MetricMap {
	flMap := make(MetricMap)
	var output map[string]interface{}
	decoder := json.NewDecoder(bytes.NewBuffer(rep.body))
	if decoder == nil {
		slog.Error("JSON decoder not iniatilized")
		return flMap
	}

	if err := decoder.Decode(&output); err != nil {
		slog.Error("Error while decoding json", "error", err)
		return flMap
	}

	addFields(&flMap, "", output)

	return flMap
}

func addFields(toMap *MetricMap, basename string, source map[string]interface{}) {
	prefix := ""
	if basename != "" {
		prefix = basename + "."
	}
	for k, v := range source {
		switch value := v.(type) {
		case float64:
			(*toMap)[prefix+k] = value
		case []interface{}:
			(*toMap)[prefix+k+"_len"] = float64(len(value))
		case map[string]interface{}:
			addFields(toMap, prefix+k, value)
		case bool:
			if value {
				(*toMap)[prefix+k] = 1
			} else {
				(*toMap)[prefix+k] = 0
			}
		}
	}
}

func (rep *rabbitJSONReply) GetString(key string) (string, bool) {
	if rep.keys == nil {
		keys := make(map[string]interface{})
		decoder := json.NewDecoder(bytes.NewBuffer(rep.body))
		if decoder == nil {
			slog.Error("JSON decoder not iniatilized")
			return "", false
		}
		err := decoder.Decode(&keys)
		if err != nil {
			return "", false
		}
		rep.keys = keys
	}
	val, ok := rep.keys[key]
	if !ok {
		return "", false
	}
	value, ok := val.(string)
	return value, ok
}

func parseTime(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err == nil {
		return t, nil
	}
	t, err = time.Parse("2006-01-02T15:04:05.999-07:00", s)
	if err == nil {
		return t, nil
	}
	return t, fmt.Errorf("time format does not match expectations")

}
