package main

import (
	"io/ioutil"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestStatsEquivalence(t *testing.T) {
	endpoints := []string{"queues", "exchanges", "nodes"}
	labels := map[string][]string{
		"queues":    queueLabelKeys,
		"exchanges": exchangeLabelKeys,
		"nodes":     nodeLabelKeys,
	}
	versions := []string{"3.6.8", "3.7.0"}
	for _, version := range versions {
		for _, endpoint := range endpoints {
			base := endpoint + "-" + version
			assertBertStatsEquivalence(t, base, labels[endpoint])
		}
	}
}

func TestMetricMapEquivalence(t *testing.T) {
	endpoints := []string{"overview"}
	versions := []string{"3.6.8", "3.7.0"}
	for _, version := range versions {
		for _, endpoint := range endpoints {
			base := endpoint + "-" + version
			assertBertMetricMapEquivalence(t, base)
		}
	}
}

func tryReadFiles(t *testing.T, base, firstExt, secondExt string) ([]byte, []byte) {
	firstFile := "testdata/" + base + "." + firstExt
	first, err := ioutil.ReadFile(firstFile)
	if err != nil {
		t.Fatalf("Error reading %s", firstFile)
	}

	secondFile := "testdata/" + base + "." + secondExt
	second, err := ioutil.ReadFile(secondFile)
	if err != nil {
		t.Fatalf("Error reading %s", secondFile)
	}
	return first, second
}

func assertBertStatsEquivalence(t *testing.T, baseFileName string, labels []string) {
	t.Helper()
	json, bert := tryReadFiles(t, baseFileName, "json", "bert")

	jsonReply := makeJSONReply(json)
	bertReply := makeBERTReply(bert)

	bertParsed := bertReply.MakeStatsInfo(labels)
	jsonParsed := jsonReply.MakeStatsInfo(labels)

	if diff := pretty.Compare(jsonParsed, bertParsed); diff != "" {
		t.Errorf("JSON/BERT mismatch for %s:\n%s", baseFileName, diff)
	}
}

func assertBertMetricMapEquivalence(t *testing.T, baseFileName string) {
	json, bert := tryReadFiles(t, baseFileName, "json", "bert")

	jsonReply := makeJSONReply(json)
	bertReply := makeBERTReply(bert)

	bertParsed := bertReply.MakeMap()
	jsonParsed := jsonReply.MakeMap()

	if diff := pretty.Compare(jsonParsed, bertParsed); diff != "" {
		t.Errorf("JSON/BERT mismatch for %s:\n%s", baseFileName, diff)
	}
}
