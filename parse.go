package ini

import (
	"fmt"
	"bufio"
	"bytes"
	"strings"
	"regexp"
	"reflect"
	"encoding/json"
)

// Encode
func Encode(v interface{}) (out []byte, err error) {

	return
}

// Decode
func Decode(blob []byte, v interface{}) (err error) {
	rv := reflect.ValueOf(v)

	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("ini: Decode of non-pointer %s", reflect.TypeOf(v))
	}

	if rv.IsNil() {
		return fmt.Errorf("ini: Decode of nil %s", reflect.TypeOf(v))
	}

	p, err := parse(string(blob))
	if err != nil {
		return
	}

	bs, err := json.Marshal(p.sections)
	if err != nil {
		return
	}

	err = json.Unmarshal(bs, v)
	return
}

// ErrSyntax is returned when there is a syntax error in an INI file.
type errSyntax struct {
	Line   int
	Source string // The contents of the erroneous line, without leading or trailing whitespace
}

func (e errSyntax) Error() string {
	return fmt.Sprintf("invalid INI syntax on line %d: %s", e.Line, e.Source)
}

var (
	// [section]
	sectionRegex   = regexp.MustCompile(`^\[(.*)\]$`)
	// foo[] = val
	assignArrRegex = regexp.MustCompile(`^([^=\[\]]+)\[\][^=]*=(.*)$`)
	// key = val
	assignRegex    = regexp.MustCompile(`^([^=]+)=(.*)$`)
	quotesRegex    = regexp.MustCompile(`^(['"])(.*)(['"])$`)
)

// parser
type parser struct {
	// for full parse(allow array, map section)
	mapping map[string]interface{}
	// for simple parse(section only allow map[string]string)
	sections map[string]Section

	// only for full parse
	NoDefSection bool
	DefSection string
	IgnoreCase bool
}

func newParser() *parser {
	return &parser{
		sections: make(map[string]Section),
	}
}

func parse(data string) (p *parser, err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			if err, ok = r.(errSyntax); ok {
				return
			}
			panic(r)
		}
	}()

	p = &parser{
		sections: make(map[string]Section),
	}

	buf := &bytes.Buffer{}
	buf.WriteString(data)

	scanner := bufio.NewScanner(buf)
	_, err = p.parse(scanner)

	return
}

func (p *parser) parseBytes(data []byte) error {
	buf := &bytes.Buffer{}
	buf.Write(data)

	scanner := bufio.NewScanner(buf)
	_, err := p.parse(scanner)

	return err
}

func (p *parser) parseString(data string) error {
	buf := &bytes.Buffer{}
	buf.WriteString(data)

	scanner := bufio.NewScanner(buf)
	_, err := p.parse(scanner)

	return err
}

// from github.com/dombenson/go-ini
func (p *parser) parse(in *bufio.Scanner) (bytes int64, err error) {
	section := DefSection
	lineNum := 0
	bytes = -1
	readLine := true

	for readLine = in.Scan(); readLine; readLine = in.Scan() {
		line := in.Text()
		bytes++
		bytes += int64(len(line))
		lineNum++
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			// Skip blank lines
			continue
		}
		if line[0] == ';' || line[0] == '#' {
			// Skip comments
			continue
		}

		// skip array parse
		if groups := assignArrRegex.FindStringSubmatch(line); groups != nil {
			// key, val := groups[1], groups[2]
			// key, val = strings.TrimSpace(key), trimWithQuotes(val)
			// curVal, ok := file.section(section).arrayValues[key]
			// if ok {
			// 	file.section(section).arrayValues[key] = append(curVal, val)
			// } else {
			// 	file.section(section).arrayValues[key] = make([]string, 1, 4)
			// 	file.section(section).arrayValues[key][0] = val
			// }
		} else if groups := assignRegex.FindStringSubmatch(line); groups != nil {
			key, val := groups[1], groups[2]
			key, val = strings.TrimSpace(key), trimWithQuotes(val)
			// file.section(section).stringValues[key] = val

			p.sections[section] = p.addToSection(section, key, val)
		} else if groups := sectionRegex.FindStringSubmatch(line); groups != nil {
			name := strings.TrimSpace(groups[1])
			section = name
			// Create the section if it does not exist
			// file.section(section)
		} else {
			err = errSyntax{lineNum, line}
			return
		}
	}

	if bytes < 0 {
		bytes = 0
	}

	err = in.Err()

	return
}

func (p *parser) addToSection(name string, key, val string) Section {
	if p.IgnoreCase {
		name = strings.ToLower(name)
		key = strings.ToLower(key)
	}

	if sec, ok := p.sections[name]; ok {
		sec[key] = val
		return sec
	}

	// create the section if it does not exist
	return Section{key: val}
}

func trimWithQuotes(inputVal string) (ret string) {
	ret = strings.TrimSpace(inputVal)
	groups := quotesRegex.FindStringSubmatch(ret)

	if groups != nil {
		if groups[1] == groups[3] {
			ret = groups[2]
		}
	}
	return
}
