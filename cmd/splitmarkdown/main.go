package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	fileutil "github.com/projectdiscovery/utils/file"
	stringsutil "github.com/projectdiscovery/utils/strings"
	"github.com/tarunKoyalwar/dsutils"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var defaultDataset = DataSet{
	Chunks: []Chunk{},
}

type DataSet struct {
	Chunks []Chunk `json:"chunks"`
}

// WriteToFile writes the dataset to a file
func (d *DataSet) WriteToFile(filename string) error {
	bin, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, bin, 0644)
}

type Chunk struct {
	Filename string                 `json:"filename"`
	Metadata map[string]interface{} `json:"metadata"`
	Content  string                 `json:"content"`
}

// SplitMarkdown splits a Markdown file/folder into chunks and returns a json file with the chunks

var (
	MarkdownInput string // can be a file or a folder
	Shape         bool   // return the structure of the markdown file (ast dump)
)

func main() {
	flag.StringVar(&MarkdownInput, "input", "", "Markdown file or folder to split")
	flag.BoolVar(&Shape, "shape", false, "Return the structure/shape of the markdown file(ast dump)")
	flag.Parse()

	if MarkdownInput == "" || !fileutil.FileOrFolderExists(MarkdownInput) {
		// Split all markdown files in the folder
		flag.Usage()
		return
	}

	switch {
	case fileutil.FileExists(MarkdownInput):
		fmt.Printf("[+] Processing %v\n", MarkdownInput)
		processFileCallback(MarkdownInput)
	case fileutil.FolderExists(MarkdownInput):
		filepath.WalkDir(MarkdownInput, func(path string, d os.DirEntry, err error) error {
			if d.IsDir() || filepath.Ext(path) != ".md" || err != nil {
				return nil
			}
			fmt.Printf("[+] Processing %v\n", path)
			err = processFileCallback(path)
			if err != nil {
				fmt.Printf("[-] Error processing %v: %v\n", path, err)
			}
			return nil
		})
	}

	if err := defaultDataset.WriteToFile("dataset.json"); err != nil {
		fmt.Printf("[-] Error writing dataset to file: %v\n", err)
	}
}

/*
goldmark.WithParserOptions(

		parser.WithAutoHeadingID(),
		parser.WithASTTransformers(),
		parser.WithBlockParsers(),
		parser.WithInlineParsers(),
		parser.WithParagraphTransformers(),
		parser.WithEscapedSpace(),
		parser.WithHeadingAttribute(),
	)
*/
func processFileCallback(filename string) error {
	// Read the file and split it
	// Write the file to the output folder
	bin, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	vector := dsutils.NewStack[string]()
	vector.Push(filepath.Base(filename))

	mdParser := parser.NewParser(
		parser.WithBlockParsers(parser.DefaultBlockParsers()...),
		parser.WithInlineParsers(parser.DefaultInlineParsers()...),
		parser.WithParagraphTransformers(parser.DefaultParagraphTransformers()...),
	)

	rootNode := mdParser.Parse(text.NewReader(bin))

	if Shape {
		rootNode.Dump(bin, 0)
		return nil
	}

	chunks := dsutils.NewStack[string]()

	SeperatorKind := ast.KindHeading

	// if markdown file does not contain any heading, use paragraph as seperator
	if !containsHeading(rootNode) {
		SeperatorKind = ast.KindParagraph
	}

	// isChunkEmpty := true
	var buff bytes.Buffer

	lastHeader := ""
	lastLevel := 0

	// walk the markdown tree and chunk it
	err = ast.Walk(rootNode, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			// fmt.Printf("entering: %v\n", node.Kind().String())
			switch node.Kind() {
			case ast.KindDocument:
				return ast.WalkContinue, nil
			case SeperatorKind:
				got := DumpHelper(node, bin, 0, nil, nil)
				var currentLevel int = 1
				if val, ok := node.(*ast.Heading); ok {
					currentLevel = val.Level
				}
				if currentLevel == 1 || currentLevel == 2 {
					currentLevel = 1
				}
				if currentLevel > lastLevel || lastLevel == 0 {
					vector.Push(lastHeader)
				} else {
					// edgecase where currentheader is example
					if stringsutil.ContainsAnyI(got, "example") {
						vector.Push(lastHeader)
					} else {
						if vector.Len() > 1 {
							// do not pop the first element
							vector.Pop()
						}
					}
				}
				vector.Push(got)
				// isChunkEmpty = true
				if buff.Len() > 0 {
					chunks.Push(buff.String())
					buff.Reset()
				}
				buff.WriteString(fmt.Sprintf("\n%v\n", vec2Markdown(vector)))
				lastHeader = got
				lastLevel = currentLevel

			case ast.KindParagraph:
				// if isChunkEmpty {
				// 	isChunkEmpty = false
				// 	buff.WriteString(vec2Markdown(vector))
				// }
				got := DumpHelper(node, bin, 0, nil, nil)
				// fmt.Printf("got: %v\n", got)
				buff.WriteString(got)

			case ast.KindFencedCodeBlock:
				// if isChunkEmpty {
				// 	isChunkEmpty = false
				// 	buff.WriteString(vec2Markdown(vector))
				// }
				n := node.(*ast.FencedCodeBlock)
				buff.WriteString(fmt.Sprintf("\n\n```%v\n", string(n.Language(bin))))
				got := DumpHelper(node, bin, 0, nil, nil)
				// fmt.Printf("got: %v\n", got)
				buff.WriteString(got)

			case ast.KindCodeBlock:
				got := DumpHelper(node, bin, 0, nil, nil)
				buff.WriteString(fmt.Sprintf("\n```\n%v\n", got))

			case ast.KindHTMLBlock:
				// if isChunkEmpty {
				// 	isChunkEmpty = false
				// 	buff.WriteString(vec2Markdown(vector))
				// }
				if value, ok := node.AttributeString("alt"); ok {
					fmt.Printf("alt: %v\n", value)
					buff.WriteString(fmt.Sprintf("%v\n", value))
				}
			case ast.KindListItem:
				// if isChunkEmpty {
				// 	isChunkEmpty = false
				// 	buff.WriteString(vec2Markdown(vector))
				// }
				buff.WriteString(fmt.Sprintf("%v\n", string(node.Text(bin))))

			}
		} else {
			// fmt.Printf("leaving: %v\n", node.Kind().String())
			switch node.Kind() {
			case SeperatorKind:
				if vector.Len() > 1 {
					// do not pop the first element
					vector.Pop()
				}
			case ast.KindFencedCodeBlock:
				buff.WriteString(fmt.Sprintln("```"))
			case ast.KindCodeBlock:
				buff.WriteString(fmt.Sprintln("```"))
			}
		}

		return ast.WalkContinue, nil
	})
	if buff.Len() > 0 {
		chunks.Push(buff.String())
		buff.Reset()
	}

	for _, v := range chunks.Vector() {
		v = strings.TrimSpace(v)
		if hasContent(v) {
			chunk := Chunk{
				Content:  v,
				Metadata: map[string]interface{}{"path": filename},
				Filename: filepath.Base(filename),
			}
			defaultDataset.Chunks = append(defaultDataset.Chunks, chunk)
		}
	}

	return err
}

// returns true if the document contains at least one heading
func containsHeading(rootNode ast.Node) bool {
	containsHeading := false
	_ = ast.Walk(rootNode, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if node.Kind() == ast.KindHeading {
			containsHeading = true
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return containsHeading
}

// func nodeCallback(node ast.Node, data []byte, vec *dsutils.Stack[string], buff *bytes.Buffer) {
// 	switch node.Kind() {
// 	case ast.KindHeading:
// 		// if it is a heading, push it to the stack
// 		vec.Push(string(node.Text(data)))
// 		buff.WriteString(vec2Markdown(vec))

// 	case ast.KindHTMLBlock:
// 		if value, ok := node.AttributeString("alt"); ok {
// 			fmt.Printf("alt: %v\n", value)
// 			buff.WriteString(fmt.Sprintf("%v\n", value))
// 		}
// 	case ast.KindFencedCodeBlock:
// 		// n := node.(*ast.FencedCodeBlock)

// 	}
// 	if node.Kind() == ast.KindHTMLBlock {
// 		// if it is a html block open it
// 		if !node.HasChildren() {
// 			out := node.Text(data)
// 			if bytes.TrimSpace(out) != nil {
// 				fmt.Printf("non-empty html blocks:\n%s\n", string(out))
// 			}
// 		} else {
// 			// if it has children, iterate over them
// 			child := node.FirstChild()
// 			for child.NextSibling() != nil {
// 				nodeCallback(child, data, vec, buff)
// 				child = child.NextSibling()
// 			}
// 		}
// 	} else {
// 		var out []byte
// 		if node.Kind() != ast.KindFencedCodeBlock && node.Kind() != ast.KindCodeBlock {
// 			out = node.Text(data)
// 		} else {
// 			// fmt.Printf("got fenced block")
// 			node.Dump(data, 1)
// 		}
// 		if bytes.TrimSpace(out) != nil {
// 			fmt.Printf("%v:\n%s\n", node.Kind().String(), string(out))
// 		}
// 		if node.HasChildren() && node.Kind() != ast.KindParagraph {
// 			// if it has children, iterate over them
// 			child := node.FirstChild()
// 			for child.NextSibling() != nil {
// 				nodeCallback(child, data, vec, buff)
// 				child = child.NextSibling()
// 			}
// 		}
// 	}
// }

func hasContent(s string) bool {
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			return true
		}
	}
	return false
}

func vec2Markdown(vector *dsutils.Stack[string]) string {
	var buff bytes.Buffer
	buff.WriteRune('\n')
	for k, v := range vector.Vector() {
		if v == "" {
			continue
		}
		buff.WriteString(fmt.Sprintf("%v. %v\n", getMultiples("#", k+1), v))
	}
	return buff.String()
}

func getMultiples(tmp string, count int) string {
	var out string
	for i := 0; i < count; i++ {
		out += tmp
	}
	return out
}

func DumpHelper(v ast.Node, source []byte, level int, kv map[string]string, cb func(int)) string {
	var buff bytes.Buffer
	if v.Type() == ast.TypeBlock {
		for i := 0; i < v.Lines().Len(); i++ {
			line := v.Lines().At(i)
			buff.WriteString(string(line.Value(source)))
		}
	}
	if cb != nil {
		cb(level + 1)
	}
	for c := v.FirstChild(); c != nil; c = c.NextSibling() {
		val := DumpHelper(c, source, level+1, kv, cb)
		buff.WriteString(val)
	}
	return buff.String()
}
