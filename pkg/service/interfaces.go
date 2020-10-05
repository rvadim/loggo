package service

type IParser interface {
	GetProperty(key string) interface{}
	ParseLine(line string) (string, error)
}
