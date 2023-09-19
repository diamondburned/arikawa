package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"unicode"
)

var (
	pkg = "gateway"
	out = "-"
)

type registry struct {
	PackageName string
	EventTypes  []EventType
}

type EventType struct {
	StructName string
	EventName  string
	IsDispatch bool
	OpCode     int
}

func (t *EventType) MethodRecv() string {
	if len(t.StructName) == 0 {
		return "e"
	}
	return string(unicode.ToLower([]rune(t.StructName)[0]))
}

//go:embed template.tmpl
var packageTmpl string

var tmpl = template.Must(template.New("").Parse(packageTmpl))

const eventStructRegex = "(?m)" +
	`^// ([A-Za-z]+(?:Event|Command)) is (a dispatch event|an event|a command)` +
	`(?:` +
	` for ([A-Z_]+)` + "|" +
	` for Op (\d+)` +
	`)?` +
	`\.(?:.|\n)*?\ntype ([A-Za-z]+(?:Event|Command)) .*`

func main() {
	flag.StringVar(&pkg, "p", pkg, "the package name to use")
	flag.StringVar(&out, "o", out, "output file, - for stdout")
	flag.Parse()

	log.Println(eventStructRegex)

	r := registry{
		PackageName: pkg,
	}

	files, err := os.ReadDir(".")
	if err != nil {
		log.Fatalln("failed to read current directory:", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") {
			continue
		}
		if err := r.CrawlFile(file.Name()); err != nil {
			log.Fatalln("failed to crawl file:", err)
		}
	}

	buf := bytes.Buffer{}
	if err := tmpl.Execute(&buf, &r); err != nil {
		log.Fatalln("failed to execute template:", err)
	}

	b, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalln("failed to fmt:", err)
	}

	output := os.Stdout
	if out != "-" {
		f, err := os.Create(out)
		if err != nil {
			log.Fatalln("failed to create output:", err)
		}
		defer f.Close()

		output = f
	}

	if _, err := output.Write(b); err != nil {
		log.Fatalln("failed to write rendered:", err)
	}
}

var reEventStruct = regexp.MustCompile(eventStructRegex)

func (r *registry) CrawlFile(name string) error {
	f, err := os.ReadFile(name)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	for _, match := range reEventStruct.FindAllSubmatch(f, -1) {
		// Validity check.
		if string(match[1]) != string(match[5]) {
			continue
		}

		if strings.HasSuffix(string(match[1]), "Command") && string(match[2]) != "a command" {
			log.Println(string(match[1]), "has invalid comment %q", string(match[2]))
			continue
		}

		t := EventType{
			StructName: string(match[1]),
			EventName:  string(match[3]),
			IsDispatch: string(match[2]) == "a dispatch event",
			OpCode:     -1,
		}

		if op := string(match[4]); op != "" && !t.IsDispatch {
			i, err := strconv.Atoi(op)
			if err != nil {
				log.Printf("error at struct %s: error parsing Op %v", t.StructName, err)
			}
			t.OpCode = i
		}

		if t.IsDispatch && t.EventName == "" {
			t.EventName = guessEventName(t.StructName)
		}

		r.EventTypes = append(r.EventTypes, t)
	}

	return nil
}

func guessEventName(structName string) string {
	name := strings.TrimSuffix(structName, "Event")

	var newName strings.Builder
	newName.Grow(len(name) * 2)

	for i, r := range name {
		if unicode.IsLower(r) {
			newName.WriteRune(unicode.ToUpper(r))
			continue
		}

		if i > 0 {
			newName.WriteByte('_')
		}

		newName.WriteRune(r)
	}

	return newName.String()
}
