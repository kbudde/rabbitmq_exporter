package main

import (
	"encoding/json"

	log "github.com/Sirupsen/logrus"
)

//MetricMap maps name to float64 metric
type MetricMap map[string]float64

//StatsInfo describes  one statistic (queue or exchange): its name, vhost it belongs to, and all associated metrics.
type StatsInfo struct {
	name    string
	vhost   string
	metrics MetricMap
}

//MakeStatsInfo creates a slice of StatsInfo from json input. Only keys with float values are mapped into `metrics`.
func MakeStatsInfo(d *json.Decoder) []StatsInfo {
	var statistics []StatsInfo
	var jsonArr []map[string]interface{}

	if d == nil {
		log.Error("JSON decoder not iniatilized")
		return make([]StatsInfo, 0)
	}

	if err := d.Decode(&jsonArr); err != nil {
		log.WithField("error", err).Error("Error while decoding json")
		return make([]StatsInfo, 0)
	}
	for _, el := range jsonArr {
		log.WithFields(log.Fields{"element": el, "vhost": el["vhost"], "name": el["name"]}).Debug("Iterate over array")
		if name, ok := el["name"]; ok {
			statsinfo := StatsInfo{}
			statsinfo.name = name.(string)
			if vhost, ok := el["vhost"]; ok {
				statsinfo.vhost = vhost.(string)
			}
			statsinfo.metrics = make(MetricMap)
			addFields(&statsinfo.metrics, "", el)
			statistics = append(statistics, statsinfo)
		}
	}

	return statistics
}

//MakeMap creates a map from json input. Only keys with float values are mapped.
func MakeMap(d *json.Decoder) MetricMap {
	flMap := make(MetricMap)
	var output map[string]interface{}

	if d == nil {
		log.Error("JSON decoder not iniatilized")
		return flMap
	}

	if err := d.Decode(&output); err != nil {
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
		case map[string]interface{}:
			addFields(toMap, prefix+k, value)
		}
	}
}
