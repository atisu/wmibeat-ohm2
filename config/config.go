// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	Wmibeat WmibeatConfig `config:"wmibeat-ohm2"`
}

type WmibeatConfig struct {
	Period     time.Duration `yaml:"period"`
	Classes    []ClassConfig
	Namespaces []NamespaceConfig `config:"namespaces"`
}

type ClassConfig struct {
	Class       string   `config:"class"`
	Fields      []string `config:"fields"`
	WhereClause string   `config:"whereclause"`
	ObjectTitle string   `config:"objecttitlecolumn"`
}

type NamespaceConfig struct {
	Namespace                string   `config:"namespace"`
	Class                    string   `config:"class"`
	MetricNameCombinedFields []string `config:"metric_name_combined_fields"`
	MetricValueField         string   `config:"metric_value_field"`
	WhereClause              string   `config:"whereclause"`
}

var DefaultConfig = WmibeatConfig{
	Period:     1 * time.Second,
	Classes:    nil,
	Namespaces: nil,
}
