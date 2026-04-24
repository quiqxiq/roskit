package command

import (
	"fmt"
	"time"
)

func BuildStreamSentence(meta *CommandMeta) []string {
	path := "/" + meta.Path
	sentence := []string{path}
	if meta.SupportsFollow {
		sentence = append(sentence, "=follow")
	}
	if meta.SupportsInterval {
		sentence = append(sentence, "=interval=1s")
	}
	return sentence
}

func BuildMonitorSentence(meta *CommandMeta, extraParams map[string]string) []string {
	path := "/" + meta.Path
	sentence := []string{path}
	for k, v := range extraParams {
		sentence = append(sentence, fmt.Sprintf("=%s=%s", k, v))
	}
	if meta.SupportsInterval {
		sentence = append(sentence, "=interval=1s")
	}
	return sentence
}

func BuildPollSentence(meta *CommandMeta) []string {
	path := "/" + meta.Path
	return []string{path}
}

func BuildQuerySentence(meta *CommandMeta, filters []string) []string {
	path := "/" + meta.Path
	sentence := []string{path}
	sentence = append(sentence, filters...)
	return sentence
}

func BuildAddSentence(path string, params map[string]string) []string {
	sentence := []string{fmt.Sprintf("/%s", path)}
	for k, v := range params {
		sentence = append(sentence, fmt.Sprintf("=%s=%s", k, v))
	}
	return sentence
}

func BuildSetSentence(path, id string, params map[string]string) []string {
	sentence := []string{fmt.Sprintf("/%s", path), fmt.Sprintf("=.id=%s", id)}
	for k, v := range params {
		sentence = append(sentence, fmt.Sprintf("=%s=%s", k, v))
	}
	return sentence
}

func BuildRemoveSentence(path, id string) []string {
	return []string{fmt.Sprintf("/%s/remove", path), fmt.Sprintf("=.id=%s", id)}
}

func BuildEnableSentence(path, id string) []string {
	return []string{fmt.Sprintf("/%s/enable", path), fmt.Sprintf("=numbers=%s", id)}
}

func BuildDisableSentence(path, id string) []string {
	return []string{fmt.Sprintf("/%s/disable", path), fmt.Sprintf("=numbers=%s", id)}
}

func PollIntervalOrDefault(meta *CommandMeta, defaultInterval time.Duration) time.Duration {
	if meta.PollInterval > 0 {
		return meta.PollInterval
	}
	return defaultInterval
}
