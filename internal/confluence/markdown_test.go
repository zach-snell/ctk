package confluence_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zach-snell/ctk/internal/confluence"
)

// update is the shared golden-file flag for the package.
var update = flag.Bool("update", false, "update .golden files")

// ---------------------------------------------------------------------------
// Table-driven unit tests
// ---------------------------------------------------------------------------

func TestMarkdownToStorage_EmptyInput(t *testing.T) {
	t.Parallel()

	if got := confluence.MarkdownToStorage(""); got != "" {
		t.Errorf("MarkdownToStorage(%q) = %q, want %q", "", got, "")
	}
}

func TestMarkdownToStorage_Headings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "h1", input: "# Title", want: "<h1>Title</h1>"},
		{name: "h2", input: "## Subtitle", want: "<h2>Subtitle</h2>"},
		{name: "h3", input: "### Section", want: "<h3>Section</h3>"},
		{name: "h4", input: "#### Sub-section", want: "<h4>Sub-section</h4>"},
		{name: "h5", input: "##### Minor", want: "<h5>Minor</h5>"},
		{name: "h6", input: "###### Tiny", want: "<h6>Tiny</h6>"},
		{
			name:  "heading with inline bold",
			input: "## **Bold** Heading",
			want:  "<h2><strong>Bold</strong> Heading</h2>",
		},
		{
			name:  "multiple headings",
			input: "# First\n\n## Second",
			want:  "<h1>First</h1><h2>Second</h2>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := confluence.MarkdownToStorage(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MarkdownToStorage(%q) mismatch (-want +got):\n%s", tt.input, diff)
			}
		})
	}
}

func TestMarkdownToStorage_InlineFormatting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bold",
			input: "some **bold** text",
			want:  "<p>some <strong>bold</strong> text</p>",
		},
		{
			name:  "strikethrough",
			input: "some ~~deleted~~ text",
			want:  "<p>some <del>deleted</del> text</p>",
		},
		{
			name:  "inline code",
			input: "use `fmt.Println()` here",
			want:  "<p>use <code>fmt.Println()</code> here</p>",
		},
		{
			name:  "link",
			input: "[Google](https://google.com)",
			want:  `<p><a href="https://google.com">Google</a></p>`,
		},
		{
			name:  "bold and code combined",
			input: "**bold** and `code`",
			want:  "<p><strong>bold</strong> and <code>code</code></p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := confluence.MarkdownToStorage(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MarkdownToStorage(%q) mismatch (-want +got):\n%s", tt.input, diff)
			}
		})
	}
}

func TestMarkdownToStorage_CodeBlocks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "fenced code with language",
			input: "```go\nfmt.Println(\"hello\")\n```",
			want: `<ac:structured-macro ac:name="code">` +
				`<ac:parameter ac:name="language">go</ac:parameter>` +
				`<ac:plain-text-body><![CDATA[fmt.Println("hello")]]></ac:plain-text-body>` +
				`</ac:structured-macro>`,
		},
		{
			name:  "fenced code without language",
			input: "```\nplain code\n```",
			want:  "<pre><code>plain code</code></pre>",
		},
		{
			name:  "fenced code with xml special chars and no lang",
			input: "```\na < b && c > d\n```",
			want:  "<pre><code>a &lt; b &amp;&amp; c &gt; d</code></pre>",
		},
		{
			name:  "multiline code block with language",
			input: "```python\ndef hello():\n    print(\"world\")\n```",
			want: `<ac:structured-macro ac:name="code">` +
				`<ac:parameter ac:name="language">python</ac:parameter>` +
				`<ac:plain-text-body><![CDATA[def hello():` + "\n" + `    print("world")]]></ac:plain-text-body>` +
				`</ac:structured-macro>`,
		},
		{
			name:  "empty code block with language",
			input: "```js\n```",
			want: `<ac:structured-macro ac:name="code">` +
				`<ac:parameter ac:name="language">js</ac:parameter>` +
				`<ac:plain-text-body><![CDATA[]]></ac:plain-text-body>` +
				`</ac:structured-macro>`,
		},
		{
			name:  "empty code block without language",
			input: "```\n```",
			want:  "<pre><code></code></pre>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := confluence.MarkdownToStorage(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MarkdownToStorage(%q) mismatch (-want +got):\n%s", tt.input, diff)
			}
		})
	}
}

func TestMarkdownToStorage_Lists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "unordered list with dashes",
			input: "- alpha\n- beta\n- gamma",
			want:  "<ul><li>alpha</li><li>beta</li><li>gamma</li></ul>",
		},
		{
			name:  "unordered list with asterisks",
			input: "* one\n* two",
			want:  "<ul><li>one</li><li>two</li></ul>",
		},
		{
			name:  "ordered list",
			input: "1. first\n2. second\n3. third",
			want:  "<ol><li>first</li><li>second</li><li>third</li></ol>",
		},
		{
			name:  "list item with inline formatting",
			input: "- **bold item**\n- `code item`",
			want:  "<ul><li><strong>bold item</strong></li><li><code>code item</code></li></ul>",
		},
		{
			name:  "single unordered item",
			input: "- only one",
			want:  "<ul><li>only one</li></ul>",
		},
		{
			name:  "single ordered item",
			input: "1. only one",
			want:  "<ol><li>only one</li></ol>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := confluence.MarkdownToStorage(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MarkdownToStorage(%q) mismatch (-want +got):\n%s", tt.input, diff)
			}
		})
	}
}

func TestMarkdownToStorage_Tables(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple two-column table",
			input: "| A | B |\n|---|---|\n| 1 | 2 |",
			want:  "<table><tbody><tr><th>A</th><th>B</th></tr><tr><td>1</td><td>2</td></tr></tbody></table>",
		},
		{
			name:  "table with inline formatting",
			input: "| Name | Value |\n|------|-------|\n| **bold** | `code` |",
			want:  "<table><tbody><tr><th>Name</th><th>Value</th></tr><tr><td><strong>bold</strong></td><td><code>code</code></td></tr></tbody></table>",
		},
		{
			name:  "table header only",
			input: "| H1 | H2 |\n|----|----|",
			want:  "<table><tbody><tr><th>H1</th><th>H2</th></tr></tbody></table>",
		},
		{
			name:  "three column table with multiple rows",
			input: "| A | B | C |\n|---|---|---|\n| 1 | 2 | 3 |\n| 4 | 5 | 6 |",
			want: "<table><tbody>" +
				"<tr><th>A</th><th>B</th><th>C</th></tr>" +
				"<tr><td>1</td><td>2</td><td>3</td></tr>" +
				"<tr><td>4</td><td>5</td><td>6</td></tr>" +
				"</tbody></table>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := confluence.MarkdownToStorage(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MarkdownToStorage(%q) mismatch (-want +got):\n%s", tt.input, diff)
			}
		})
	}
}

func TestMarkdownToStorage_Blockquotes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single line blockquote",
			input: "> This is a quote",
			want:  "<blockquote><p>This is a quote</p></blockquote>",
		},
		{
			name:  "multi-line blockquote",
			input: "> Line one\n> Line two",
			want:  "<blockquote><p>Line one Line two</p></blockquote>",
		},
		{
			name:  "blockquote with inline formatting",
			input: "> **bold** quote",
			want:  "<blockquote><p><strong>bold</strong> quote</p></blockquote>",
		},
		{
			name:  "blockquote with empty marker",
			input: ">bare text",
			want:  "<blockquote><p>bare text</p></blockquote>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := confluence.MarkdownToStorage(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MarkdownToStorage(%q) mismatch (-want +got):\n%s", tt.input, diff)
			}
		})
	}
}

func TestMarkdownToStorage_HorizontalRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "triple dashes", input: "---", want: "<hr />"},
		{name: "triple asterisks", input: "***", want: "<hr />"},
		{name: "triple underscores", input: "___", want: "<hr />"},
		{name: "long dashes", input: "--------", want: "<hr />"},
		{
			name:  "hr between paragraphs",
			input: "above\n\n---\n\nbelow",
			want:  "<p>above</p><hr /><p>below</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := confluence.MarkdownToStorage(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MarkdownToStorage(%q) mismatch (-want +got):\n%s", tt.input, diff)
			}
		})
	}
}

func TestMarkdownToStorage_Paragraphs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single paragraph",
			input: "Hello world",
			want:  "<p>Hello world</p>",
		},
		{
			name:  "two paragraphs separated by blank line",
			input: "First paragraph\n\nSecond paragraph",
			want:  "<p>First paragraph</p><p>Second paragraph</p>",
		},
		{
			name:  "consecutive lines merge into one paragraph",
			input: "Line one\nLine two\nLine three",
			want:  "<p>Line one Line two Line three</p>",
		},
		{
			name:  "paragraph before heading",
			input: "Some text\n\n# Heading",
			want:  "<p>Some text</p><h1>Heading</h1>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := confluence.MarkdownToStorage(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MarkdownToStorage(%q) mismatch (-want +got):\n%s", tt.input, diff)
			}
		})
	}
}

func TestMarkdownToStorage_SpecialXMLCharacters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "code block escapes angle brackets and ampersand",
			input: "```\na < b && c > d\n```",
			want:  "<pre><code>a &lt; b &amp;&amp; c &gt; d</code></pre>",
		},
		{
			name:  "code block escapes quotes",
			input: "```\nkey=\"value\"\n```",
			want:  `<pre><code>key=&quot;value&quot;</code></pre>`,
		},
		{
			name:  "non-word language chars fall through",
			input: "```c++\nint x;\n```",
			// "```c++" doesn't match mdFencedCodeOpen because \w* captures "c"
			// but "++" doesn't match \s*$. The first line becomes a paragraph line,
			// "int x;" is also a paragraph line, but "```" alone matches mdFencedCodeClose
			// which is also a valid mdFencedCodeOpen (with empty lang), so it produces
			// an empty code block.
			want: "<p>```c++ int x;</p><pre><code></code></pre>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := confluence.MarkdownToStorage(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MarkdownToStorage(%q) mismatch (-want +got):\n%s", tt.input, diff)
			}
		})
	}
}

func TestMarkdownToStorage_LiteralNewlines(t *testing.T) {
	t.Parallel()

	// The converter normalizes literal \n (backslash-n from JSON) into real newlines.
	input := `# Title\n\nSome text\n- item one\n- item two`
	got := confluence.MarkdownToStorage(input)

	want := "<h1>Title</h1><p>Some text</p><ul><li>item one</li><li>item two</li></ul>"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("MarkdownToStorage(literal \\n) mismatch (-want +got):\n%s", diff)
	}
}

func TestMarkdownToStorage_WindowsLineEndings(t *testing.T) {
	t.Parallel()

	input := "# Title\r\n\r\nSome text\r\n"
	got := confluence.MarkdownToStorage(input)

	want := "<h1>Title</h1><p>Some text</p>"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("MarkdownToStorage(CRLF) mismatch (-want +got):\n%s", diff)
	}
}

func TestMarkdownToStorage_UnclosedCodeBlock(t *testing.T) {
	t.Parallel()

	// Code block that never closes — should consume remaining lines.
	input := "```go\nfunc main() {\n"
	got := confluence.MarkdownToStorage(input)

	want := `<ac:structured-macro ac:name="code">` +
		`<ac:parameter ac:name="language">go</ac:parameter>` +
		`<ac:plain-text-body><![CDATA[func main() {` + "\n" + `]]></ac:plain-text-body>` +
		`</ac:structured-macro>`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("MarkdownToStorage(unclosed code) mismatch (-want +got):\n%s", diff)
	}
}

// ---------------------------------------------------------------------------
// Golden file tests
// ---------------------------------------------------------------------------

func TestMarkdownToStorage_Golden(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("testdata/*.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no test fixtures found in testdata/")
	}

	for _, inputFile := range files {
		name := strings.TrimSuffix(filepath.Base(inputFile), ".md")
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			input, err := os.ReadFile(inputFile)
			if err != nil {
				t.Fatal(err)
			}

			got := confluence.MarkdownToStorage(string(input))

			goldenFile := strings.TrimSuffix(inputFile, ".md") + ".golden"
			if *update {
				if err := os.WriteFile(goldenFile, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}

			want, err := os.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("golden file not found (run with -update to create): %v", err)
			}
			if diff := cmp.Diff(string(want), got); diff != "" {
				t.Errorf("golden mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Fuzz test
// ---------------------------------------------------------------------------

func FuzzMarkdownToStorage(f *testing.F) {
	// Seed corpus covering every code path in the converter.
	seeds := []string{
		"",
		"plain text",
		"# Heading 1",
		"## Heading 2",
		"### Heading 3",
		"#### Heading 4",
		"##### Heading 5",
		"###### Heading 6",
		"**bold text**",
		"*italic text*",
		"~~strikethrough~~",
		"`inline code`",
		"[link](https://example.com)",
		"https://bare-url.com in text",
		"```go\nfmt.Println()\n```",
		"```\nno language\n```",
		"> blockquote line",
		"> line one\n> line two",
		"- item 1\n- item 2",
		"* item A\n* item B",
		"1. first\n2. second",
		"| a | b |\n|---|---|\n| 1 | 2 |",
		"---",
		"***",
		"___",
		"paragraph one\n\nparagraph two",
		"line one\nline two\nline three",
		"a < b && c > d",
		"key=\"value\"",
		"```\na < b && c > d\n```",
		"# Title\n\n**Bold** paragraph\n\n- list\n\n> quote\n\n---\n\n```go\ncode()\n```",
		`mixed\ncontent\nwith\nliteral\nnewlines`,
		"# Heading\r\n\r\nCRLF line\r\n",
		"```go\nunclosed code block",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// Property: must never panic.
		got := confluence.MarkdownToStorage(input)

		// Property: empty input must produce empty output.
		if input == "" && got != "" {
			t.Errorf("MarkdownToStorage(%q) = %q, want empty string", input, got)
		}

		// Property: non-empty input with visible content should produce non-empty output.
		// Some edge cases (e.g., only whitespace or newlines) legitimately produce "".
		// We intentionally do not flag these.

		// Property: output should not contain raw \r characters.
		if strings.Contains(got, "\r") {
			t.Errorf("MarkdownToStorage(%q) output contains \\r", input)
		}
	})
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkMarkdownToStorage(b *testing.B) {
	cases := []struct {
		name  string
		input string
	}{
		{name: "empty", input: ""},
		{name: "single_paragraph", input: "Just a simple paragraph of text."},
		{name: "heading", input: "## Quick Heading"},
		{name: "inline_formatting", input: "**Bold** and `code` and ~~strike~~ and [link](https://x.com)"},
		{name: "code_block", input: "```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```"},
		{name: "list", input: "- one\n- two\n- three\n- four\n- five"},
		{name: "table", input: "| A | B | C |\n|---|---|---|\n| 1 | 2 | 3 |\n| 4 | 5 | 6 |"},
		{
			name: "complex_document",
			input: "# Title\n\n" +
				"Paragraph with **bold** and `code` and [link](https://example.com).\n\n" +
				"## Section\n\n" +
				"- item one\n- item two\n- item three\n\n" +
				"1. first\n2. second\n\n" +
				"> A blockquote\n> spanning lines\n\n" +
				"---\n\n" +
				"```python\ndef hello():\n    print(\"world\")\n```\n\n" +
				"| Col A | Col B |\n|-------|-------|\n| 1     | 2     |\n| 3     | 4     |\n\n" +
				"Final paragraph.",
		},
	}

	for _, bc := range cases {
		b.Run(bc.name, func(b *testing.B) {
			for b.Loop() {
				confluence.MarkdownToStorage(bc.input)
			}
		})
	}
}
