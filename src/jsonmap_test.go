package main

import (
	"testing"
)

func TestWithInvalidJSON(t *testing.T) {
	invalidJSONReply := makeJSONReply([]byte("I'm no json"))

	if mm := invalidJSONReply.MakeMap(); mm == nil {
		t.Errorf("Json is invalid. Empty map should be returned. Value: %v", mm)
	}
	if qi := invalidJSONReply.MakeStatsInfo(queueLabelKeys); qi == nil {
		t.Errorf("Json is invalid. Empty map should be returned. Value: %v", qi)
	}
}

func checkMap(flMap map[string]float64, t *testing.T, addValue float64) {
	if flMap == nil {
		t.Error("Map should not be nil")
	}

	if v := flMap["FloatKey"]; v != addValue+4 {
		t.Errorf("Map should contain FloatKey but found '%v'", v)
	}

	if v := flMap["nes.ted"]; v != addValue+5 {
		t.Errorf("Map should contain nes.ted key but found '%v'", v)
	}

	if v, ok := flMap["st"]; ok {
		t.Errorf("key 'st' should not be included in map as it contains a string. Value: %v", v)
	}
}

func TestMakeMap(t *testing.T) {
	reply := makeJSONReply([]byte(`{"FloatKey":4, "st":"string","nes":{"ted":5}}`))

	flMap := reply.MakeMap()
	checkMap(flMap, t, 0)
}

func TestMakeStatsInfo(t *testing.T) {
	reply := makeJSONReply([]byte(`[{"name":"q1", "FloatKey":14,"nes":{"ted":15}},{"name":"q2", "vhost":"foo", "FloatKey":24,"nes":{"ted":25}}]`))

	qinfo := reply.MakeStatsInfo(queueLabelKeys)
	t.Log(qinfo)
	if qinfo[0].labels["name"] != "q1" {
		t.Errorf("unexpected qinfo name: %v", qinfo[0].labels["name"])
	}
	if qinfo[1].labels["name"] != "q2" {
		t.Errorf("unexpected qinfo name: %v", qinfo[0].labels["name"])
	}
	if qinfo[1].labels["vhost"] != "foo" {
		t.Errorf("unexpected qinfo name: %v", qinfo[0].labels["name"])
	}
	checkMap(qinfo[0].metrics, t, 10)
	checkMap(qinfo[1].metrics, t, 20)
}

func TestArraySize(t *testing.T) {
	reply := makeJSONReply([]byte(`{"node":"node1","empty_partitions": [],"partitions": [{"name":"node1"},{"name":"node2"}]}`))

	node := reply.MakeMap()
	t.Log(node)
	if v, ok := node["empty_partitions_len"]; !ok || v != 0 {
		t.Error("Unexpected (empty) partitions size", v)
	}
	if v, ok := node["partitions_len"]; !ok || v != 2 {
		t.Error("Unexpected partitions size", v)
	}
}
