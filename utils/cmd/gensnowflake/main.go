package main

import (
	"bytes"
	"flag"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"text/template"

	_ "embed"
)

type data struct {
	Package       string
	ImportDiscord bool
	Snowflakes    []snowflakeType
}

type snowflakeType struct {
	TypeName string
}

//go:embed template.tmpl
var packageTmpl string

var tmpl = template.Must(template.New("").Parse(packageTmpl))

func main() {
	var pkg string
	var out string

	log.SetFlags(0)

	flag.Usage = func() {
		log.Printf("usage: %s [-p package] <type names...>", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	flag.StringVar(&out, "o", "", "output, empty for stdout")
	flag.StringVar(&pkg, "p", "discord", "package name")
	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	d := data{
		Package:       pkg,
		ImportDiscord: pkg != "discord",
	}

	for _, arg := range flag.Args() {
		d.Snowflakes = append(d.Snowflakes, snowflakeType{
			TypeName: arg,
		})
	}

	buf := bytes.Buffer{}
	if err := tmpl.Execute(&buf, d); err != nil {
		log.Fatalln("failed to execute template:", err)
	}

	b, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalln("failed to fmt:", err)
	}

	outFile := os.Stdout

	if out != "" {
		f, err := os.Create(out)
		if err != nil {
			log.Fatalln("failed to create output file:", err)
		}
		defer f.Close()

		outFile = f
	}

	if _, err := outFile.Write(b); err != nil {
		log.Fatalln("failed to write to file:", err)
	}
}
