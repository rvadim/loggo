package containerd

// ParseLine used to parse docker log string, and try to parse inner log
import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"rvadim/loggo/pkg/parser"
	"strconv"
	"strings"
)

// Parser parse json and extend it with data
type Parser struct {
	containerdRegexp *regexp.Regexp
	properties       parser.Properties
}

// New creats new parser
func New(p parser.Properties) *Parser {
	return &Parser{
		containerdRegexp: regexp.MustCompile("(?s)^(.+) (stdout|stderr) . (.*)$"),
		properties:       p,
	}
}

// ParseLine parse JSON string and put all fields to upper json
func (p *Parser) ParseLine(line string) (string, error) {
	var i = make(map[string]interface{})
	output := p.containerdRegexp.FindStringSubmatch(line)
	if len(output) != 4 {
		return line, fmt.Errorf("unable to parse containerd line '%s'", line)
	}
	i["time"] = output[1]
	i["stream"] = output[2]
	i["log"] = output[3]
	var inner interface{}
	err := json.Unmarshal([]byte(output[3]), &inner)
	if err != nil {
		out, ierr := p.extend(i)
		if ierr != nil {
			return line, ierr
		}
		return out, err
	}

	innerMap, ok := inner.(map[string]interface{})
	if !ok {
		out, ierr := p.extend(i)
		if ierr != nil {
			return line, ierr
		}
		return out, err
	}
	for key, value := range innerMap {
		if value == nil {
			// INFO reflect.TypeOf(nil).Kind() cause panic so check nil here
			i[key] = nil
			continue
		}

		if key == "upstream_response_time" {
			var floatValue float64
			var terr = errors.New("Not nil")

			if reflect.TypeOf(value).Kind() == reflect.Float64 {
				floatValue, terr = value.(float64), nil
			}
			if reflect.TypeOf(value).Kind() == reflect.String {
				floatValue, terr = transformValue(value.(string))
			}

			if terr == nil {
				i["upstream_response_time_float"] = floatValue
			}
		}

		if reflect.TypeOf(value).Kind() == reflect.Map {
			if value, ok := value.(map[string]interface{}); ok {
				for k, v := range value {
					if v == nil {
						i[strings.Join([]string{key, k}, ".")] = nil
						continue
					}
					if reflect.TypeOf(v).Kind() == reflect.Map {
						// We don't want to parse recursive yet
						continue
					}
					i[strings.Join([]string{key, k}, ".")] = v
				}
				continue
			}
			continue
		}

		i[key] = value
	}
	delete(i, "log")
	out, err := p.extend(i)
	if err != nil {
		return line, err
	}
	return string(out), nil
}

func transformValue(value string) (float64, error) {
	value = strings.Replace(value, " ", "", -1)
	values := strings.Split(value, ",")

	lastValue := values[len(values)-1]
	return strconv.ParseFloat(lastValue, 64)
}

func (p *Parser) extend(a parser.Properties) (string, error) {
	for k, v := range p.properties {
		a[k] = v
	}
	out, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// GetProperty return specifiv extends(key=value parametes) for parsed files
// For example kubernetes.container_name or kubernetes.namespace
func (p *Parser) GetProperty(key string) interface{} {
	if val, ok := p.properties[key]; ok {
		return val
	}
	return nil
}
