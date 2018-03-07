package main

import (
	"bytes"
	"encoding/json"
	"strconv"

	log "github.com/Sirupsen/logrus"
)

type rabbitJSONReply struct {
	decoder *json.Decoder
}

func makeJSONReply(body []byte) RabbitReply {
	decoder := json.NewDecoder(bytes.NewBuffer(body))
	return &rabbitJSONReply{decoder}
}

//MakeStatsInfo creates a slice of StatsInfo from json input. Only keys with float values are mapped into `metrics`.
func (rep *rabbitJSONReply) MakeStatsInfo(labels []string) []StatsInfo {
	var statistics []StatsInfo
	var jsonArr []map[string]interface{}

	if rep.decoder == nil {
		log.Error("JSON decoder not iniatilized")
		return make([]StatsInfo, 0)
	}

	if err := rep.decoder.Decode(&jsonArr); err != nil {
		log.WithField("error", err).Error("Error while decoding json")
		return make([]StatsInfo, 0)
	}
	for _, el := range jsonArr {
		log.WithFields(log.Fields{"element": el, "vhost": el["vhost"], "name": el["name"]}).Debug("Iterate over array")
		if _, ok := el["name"]; ok {
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

	if rep.decoder == nil {
		log.Error("JSON decoder not iniatilized")
		return flMap
	}

	if err := rep.decoder.Decode(&output); err != nil {
		log.WithField("error", err).Error("Error while decoding json")
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
