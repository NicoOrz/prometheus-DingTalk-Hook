// 包 router 提供基于 receiver/status/labels 的规则匹配能力。
package router

import (
	"strings"

	"prometheus-dingtalk-hook/internal/alertmanager"
	"prometheus-dingtalk-hook/internal/config"
)

type When struct {
	receivers map[string]struct{}
	statuses  map[string]struct{}
	labels    map[string]map[string]struct{}
}

func CompileWhen(c config.WhenConfig) When {
	w := When{
		receivers: make(map[string]struct{}, len(c.Receiver)),
		statuses:  make(map[string]struct{}, len(c.Status)),
		labels:    make(map[string]map[string]struct{}, len(c.Labels)),
	}

	for _, v := range c.Receiver {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		w.receivers[v] = struct{}{}
	}

	for _, v := range c.Status {
		v = strings.TrimSpace(strings.ToLower(v))
		if v == "" {
			continue
		}
		w.statuses[v] = struct{}{}
	}

	for k, vs := range c.Labels {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		set := make(map[string]struct{}, len(vs))
		for _, v := range vs {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			set[v] = struct{}{}
		}
		if len(set) == 0 {
			continue
		}
		w.labels[k] = set
	}

	return w
}

func (w When) Match(msg alertmanager.WebhookMessage) bool {
	if len(w.receivers) > 0 {
		if _, ok := w.receivers[msg.Receiver]; !ok {
			return false
		}
	}

	if len(w.statuses) > 0 {
		status := strings.TrimSpace(strings.ToLower(msg.Status))
		if _, ok := w.statuses[status]; !ok {
			return false
		}
	}

	if len(w.labels) > 0 {
		for k, allowed := range w.labels {
			v, ok := msg.CommonLabels[k]
			if !ok {
				v, ok = msg.GroupLabels[k]
			}
			if !ok {
				return false
			}
			if _, ok := allowed[v]; !ok {
				return false
			}
		}
	}

	return true
}

type Route struct {
	Name     string
	When     When
	Channels []string
}

func CompileRoutes(routes []config.RouteConfig) []Route {
	out := make([]Route, 0, len(routes))
	for _, r := range routes {
		out = append(out, Route{
			Name:     r.Name,
			When:     CompileWhen(r.When),
			Channels: append([]string(nil), r.Channels...),
		})
	}
	return out
}

func FirstMatch(routes []Route, msg alertmanager.WebhookMessage) []string {
	for _, r := range routes {
		if r.When.Match(msg) {
			return r.Channels
		}
	}
	return nil
}

type MentionRule struct {
	Name    string
	When    When
	Mention config.MentionConfig
}

func CompileMentionRules(rules []config.MentionRuleConfig) []MentionRule {
	out := make([]MentionRule, 0, len(rules))
	for _, r := range rules {
		out = append(out, MentionRule{
			Name:    r.Name,
			When:    CompileWhen(r.When),
			Mention: r.Mention,
		})
	}
	return out
}

func MergeMention(base config.MentionConfig, extra config.MentionConfig) config.MentionConfig {
	out := base
	out.AtAll = out.AtAll || extra.AtAll
	if len(extra.AtMobiles) > 0 {
		out.AtMobiles = append(out.AtMobiles, extra.AtMobiles...)
	}
	if len(extra.AtUserIds) > 0 {
		out.AtUserIds = append(out.AtUserIds, extra.AtUserIds...)
	}
	return out
}
