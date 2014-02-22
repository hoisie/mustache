package mustache

import (
    "testing"
    "encoding/json"
    "io/ioutil"
    "os"
)

func mustDecodeJson (str string) interface{} {
    var data interface{}
    err := json.Unmarshal([]byte(str), &data)
    if(err != nil) { panic(err) }
    return data
}





// -----------------------------------------------------------------------------

// Comment blocks should be removed from the template.
func TestCommentsInline(t *testing.T) { 
    template := "12345{{! Comment Block! }}67890"
    expected := "1234567890"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Multiline comments should be permitted.
func TestCommentsMultiline(t *testing.T) { 
    template := "12345{{!\n  This is a\n  multi-line comment...\n}}67890\n"
    expected := "1234567890\n"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// All standalone comment lines should be removed.
func TestCommentsStandalone(t *testing.T) { 
    template := "Begin.\n{{! Comment Block! }}\nEnd.\n"
    expected := "Begin.\nEnd.\n"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// All standalone comment lines should be removed.
func TestCommentsIndentedStandalone(t *testing.T) { 
    template := "Begin.\n  {{! Indented Comment Block! }}\nEnd.\n"
    expected := "Begin.\nEnd.\n"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// "\r\n" should be considered a newline for standalone tags.
func TestCommentsStandaloneLineEndings(t *testing.T) { 
    template := "|\r\n{{! Standalone Comment }}\r\n|"
    expected := "|\r\n|"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Standalone tags should not require a newline to precede them.
func TestCommentsStandaloneWithoutPreviousLine(t *testing.T) { 
    template := "  {{! I'm Still Standalone }}\n!"
    expected := "!"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Standalone tags should not require a newline to follow them.
func TestCommentsStandaloneWithoutNewline(t *testing.T) { 
    template := "!\n  {{! I'm Still Standalone }}"
    expected := "!\n"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// All standalone comment lines should be removed.
func TestCommentsMultilineStandalone(t *testing.T) { 
    template := "Begin.\n{{!\nSomething's going on here...\n}}\nEnd.\n"
    expected := "Begin.\nEnd.\n"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// All standalone comment lines should be removed.
func TestCommentsIndentedMultilineStandalone(t *testing.T) { 
    template := "Begin.\n  {{!\n    Something's going on here...\n  }}\nEnd.\n"
    expected := "Begin.\nEnd.\n"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Inline comments should not strip whitespace
func TestCommentsIndentedInline(t *testing.T) { 
    template := "  12 {{! 34 }}\n"
    expected := "  12 \n"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Comment removal should preserve surrounding whitespace.
func TestCommentsSurroundingWhitespace(t *testing.T) { 
    template := "12345 {{! Comment Block! }} 67890"
    expected := "12345  67890"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}




// -----------------------------------------------------------------------------

// The equals sign (used on both sides) should permit delimiter changes.
func TestDelimitersPairBehavior(t *testing.T) { 
    template := "{{=<% %>=}}(<%text%>)"
    data     := mustDecodeJson("{\"text\":\"Hey!\"}")
    expected := "(Hey!)"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Characters with special meaning regexen should be valid delimiters.
func TestDelimitersSpecialCharacters(t *testing.T) { 
    template := "({{=[ ]=}}[text])"
    data     := mustDecodeJson("{\"text\":\"It worked!\"}")
    expected := "(It worked!)"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Delimiters set outside sections should persist.
func TestDelimitersSections(t *testing.T) { 
    template := "[\n{{#section}}\n  {{data}}\n  |data|\n{{/section}}\n\n{{= | | =}}\n|#section|\n  {{data}}\n  |data|\n|/section|\n]\n"
    data     := mustDecodeJson("{\"data\":\"I got interpolated.\",\"section\":true}")
    expected := "[\n  I got interpolated.\n  |data|\n\n  {{data}}\n  I got interpolated.\n]\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Delimiters set outside inverted sections should persist.
func TestDelimitersInvertedSections(t *testing.T) { 
    template := "[\n{{^section}}\n  {{data}}\n  |data|\n{{/section}}\n\n{{= | | =}}\n|^section|\n  {{data}}\n  |data|\n|/section|\n]\n"
    data     := mustDecodeJson("{\"data\":\"I got interpolated.\",\"section\":false}")
    expected := "[\n  I got interpolated.\n  |data|\n\n  {{data}}\n  I got interpolated.\n]\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Delimiters set in a parent template should not affect a partial.
func TestDelimitersPartialInheritence(t *testing.T) { 
    ioutil.WriteFile("include", []byte(".{{value}}."), 0666)
    defer os.Remove("include")
    
    template := "[ {{>include}} ]\n{{= | | =}}\n[ |>include| ]\n"
    data     := mustDecodeJson("{\"value\":\"yes\"}")
    expected := "[ .yes. ]\n[ .yes. ]\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Delimiters set in a partial should not affect the parent template.
func TestDelimitersPostPartialBehavior(t *testing.T) { 
    ioutil.WriteFile("include", []byte(".{{value}}. {{= | | =}} .|value|."), 0666)
    defer os.Remove("include")
    
    template := "[ {{>include}} ]\n[ .{{value}}.  .|value|. ]\n"
    data     := mustDecodeJson("{\"value\":\"yes\"}")
    expected := "[ .yes.  .yes. ]\n[ .yes.  .|value|. ]\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Surrounding whitespace should be left untouched.
func TestDelimitersSurroundingWhitespace(t *testing.T) { 
    template := "| {{=@ @=}} |"
    expected := "|  |"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Whitespace should be left untouched.
func TestDelimitersOutlyingWhitespaceInline(t *testing.T) { 
    template := " | {{=@ @=}}\n"
    expected := " | \n"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Standalone lines should be removed from the template.
func TestDelimitersStandaloneTag(t *testing.T) { 
    template := "Begin.\n{{=@ @=}}\nEnd.\n"
    expected := "Begin.\nEnd.\n"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Indented standalone lines should be removed from the template.
func TestDelimitersIndentedStandaloneTag(t *testing.T) { 
    template := "Begin.\n  {{=@ @=}}\nEnd.\n"
    expected := "Begin.\nEnd.\n"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// "\r\n" should be considered a newline for standalone tags.
func TestDelimitersStandaloneLineEndings(t *testing.T) { 
    template := "|\r\n{{= @ @ =}}\r\n|"
    expected := "|\r\n|"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Standalone tags should not require a newline to precede them.
func TestDelimitersStandaloneWithoutPreviousLine(t *testing.T) { 
    template := "  {{=@ @=}}\n="
    expected := "="
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Standalone tags should not require a newline to follow them.
func TestDelimitersStandaloneWithoutNewline(t *testing.T) { 
    template := "=\n  {{=@ @=}}"
    expected := "=\n"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Superfluous in-tag whitespace should be ignored.
func TestDelimitersPairwithPadding(t *testing.T) { 
    template := "|{{= @   @ =}}|"
    expected := "||"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}




// -----------------------------------------------------------------------------

// Mustache-free templates should render as-is.
func TestInterpolationNoInterpolation(t *testing.T) { 
    template := "Hello from {Mustache}!\n"
    expected := "Hello from {Mustache}!\n"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Unadorned tags should interpolate content into the template.
func TestInterpolationBasicInterpolation(t *testing.T) { 
    template := "Hello, {{subject}}!\n"
    data     := mustDecodeJson("{\"subject\":\"world\"}")
    expected := "Hello, world!\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Basic interpolation should be HTML escaped.
func TestInterpolationHTMLEscaping(t *testing.T) { 
    template := "These characters should be HTML escaped: {{forbidden}}\n"
    data     := mustDecodeJson("{\"forbidden\":\"\\u0026 \\\" \\u003c \\u003e\"}")
    expected := "These characters should be HTML escaped: &amp; &quot; &lt; &gt;\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Triple mustaches should interpolate without HTML escaping.
func TestInterpolationTripleMustache(t *testing.T) { 
    template := "These characters should not be HTML escaped: {{{forbidden}}}\n"
    data     := mustDecodeJson("{\"forbidden\":\"\\u0026 \\\" \\u003c \\u003e\"}")
    expected := "These characters should not be HTML escaped: & \" < >\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Ampersand should interpolate without HTML escaping.
func TestInterpolationAmpersand(t *testing.T) { 
    template := "These characters should not be HTML escaped: {{&forbidden}}\n"
    data     := mustDecodeJson("{\"forbidden\":\"\\u0026 \\\" \\u003c \\u003e\"}")
    expected := "These characters should not be HTML escaped: & \" < >\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Integers should interpolate seamlessly.
func TestInterpolationBasicIntegerInterpolation(t *testing.T) { 
    template := "\"{{mph}} miles an hour!\""
    data     := mustDecodeJson("{\"mph\":85}")
    expected := "\"85 miles an hour!\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Integers should interpolate seamlessly.
func TestInterpolationTripleMustacheIntegerInterpolation(t *testing.T) { 
    template := "\"{{{mph}}} miles an hour!\""
    data     := mustDecodeJson("{\"mph\":85}")
    expected := "\"85 miles an hour!\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Integers should interpolate seamlessly.
func TestInterpolationAmpersandIntegerInterpolation(t *testing.T) { 
    template := "\"{{&mph}} miles an hour!\""
    data     := mustDecodeJson("{\"mph\":85}")
    expected := "\"85 miles an hour!\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Decimals should interpolate seamlessly with proper significance.
func TestInterpolationBasicDecimalInterpolation(t *testing.T) { 
    template := "\"{{power}} jiggawatts!\""
    data     := mustDecodeJson("{\"power\":1.21}")
    expected := "\"1.21 jiggawatts!\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Decimals should interpolate seamlessly with proper significance.
func TestInterpolationTripleMustacheDecimalInterpolation(t *testing.T) { 
    template := "\"{{{power}}} jiggawatts!\""
    data     := mustDecodeJson("{\"power\":1.21}")
    expected := "\"1.21 jiggawatts!\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Decimals should interpolate seamlessly with proper significance.
func TestInterpolationAmpersandDecimalInterpolation(t *testing.T) { 
    template := "\"{{&power}} jiggawatts!\""
    data     := mustDecodeJson("{\"power\":1.21}")
    expected := "\"1.21 jiggawatts!\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Failed context lookups should default to empty strings.
func TestInterpolationBasicContextMissInterpolation(t *testing.T) { 
    template := "I ({{cannot}}) be seen!"
    expected := "I () be seen!"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Failed context lookups should default to empty strings.
func TestInterpolationTripleMustacheContextMissInterpolation(t *testing.T) { 
    template := "I ({{{cannot}}}) be seen!"
    expected := "I () be seen!"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Failed context lookups should default to empty strings.
func TestInterpolationAmpersandContextMissInterpolation(t *testing.T) { 
    template := "I ({{&cannot}}) be seen!"
    expected := "I () be seen!"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Dotted names should be considered a form of shorthand for sections.
func TestInterpolationDottedNamesBasicInterpolation(t *testing.T) { 
    template := "\"{{person.name}}\" == \"{{#person}}{{name}}{{/person}}\""
    data     := mustDecodeJson("{\"person\":{\"name\":\"Joe\"}}")
    expected := "\"Joe\" == \"Joe\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Dotted names should be considered a form of shorthand for sections.
func TestInterpolationDottedNamesTripleMustacheInterpolation(t *testing.T) { 
    template := "\"{{{person.name}}}\" == \"{{#person}}{{{name}}}{{/person}}\""
    data     := mustDecodeJson("{\"person\":{\"name\":\"Joe\"}}")
    expected := "\"Joe\" == \"Joe\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Dotted names should be considered a form of shorthand for sections.
func TestInterpolationDottedNamesAmpersandInterpolation(t *testing.T) { 
    template := "\"{{&person.name}}\" == \"{{#person}}{{&name}}{{/person}}\""
    data     := mustDecodeJson("{\"person\":{\"name\":\"Joe\"}}")
    expected := "\"Joe\" == \"Joe\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Dotted names should be functional to any level of nesting.
func TestInterpolationDottedNamesArbitraryDepth(t *testing.T) { 
    template := "\"{{a.b.c.d.e.name}}\" == \"Phil\""
    data     := mustDecodeJson("{\"a\":{\"b\":{\"c\":{\"d\":{\"e\":{\"name\":\"Phil\"}}}}}}")
    expected := "\"Phil\" == \"Phil\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Any falsey value prior to the last part of the name should yield ''.
func TestInterpolationDottedNamesBrokenChains(t *testing.T) { 
    template := "\"{{a.b.c}}\" == \"\""
    data     := mustDecodeJson("{\"a\":{}}")
    expected := "\"\" == \"\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Each part of a dotted name should resolve only against its parent.
func TestInterpolationDottedNamesBrokenChainResolution(t *testing.T) { 
    template := "\"{{a.b.c.name}}\" == \"\""
    data     := mustDecodeJson("{\"a\":{\"b\":{}},\"c\":{\"name\":\"Jim\"}}")
    expected := "\"\" == \"\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// The first part of a dotted name should resolve as any other name.
func TestInterpolationDottedNamesInitialResolution(t *testing.T) { 
    template := "\"{{#a}}{{b.c.d.e.name}}{{/a}}\" == \"Phil\""
    data     := mustDecodeJson("{\"a\":{\"b\":{\"c\":{\"d\":{\"e\":{\"name\":\"Phil\"}}}}},\"b\":{\"c\":{\"d\":{\"e\":{\"name\":\"Wrong\"}}}}}")
    expected := "\"Phil\" == \"Phil\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Interpolation should not alter surrounding whitespace.
func TestInterpolationInterpolationSurroundingWhitespace(t *testing.T) { 
    template := "| {{string}} |"
    data     := mustDecodeJson("{\"string\":\"---\"}")
    expected := "| --- |"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Interpolation should not alter surrounding whitespace.
func TestInterpolationTripleMustacheSurroundingWhitespace(t *testing.T) { 
    template := "| {{{string}}} |"
    data     := mustDecodeJson("{\"string\":\"---\"}")
    expected := "| --- |"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Interpolation should not alter surrounding whitespace.
func TestInterpolationAmpersandSurroundingWhitespace(t *testing.T) { 
    template := "| {{&string}} |"
    data     := mustDecodeJson("{\"string\":\"---\"}")
    expected := "| --- |"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Standalone interpolation should not alter surrounding whitespace.
func TestInterpolationInterpolationStandalone(t *testing.T) { 
    template := "  {{string}}\n"
    data     := mustDecodeJson("{\"string\":\"---\"}")
    expected := "  ---\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Standalone interpolation should not alter surrounding whitespace.
func TestInterpolationTripleMustacheStandalone(t *testing.T) { 
    template := "  {{{string}}}\n"
    data     := mustDecodeJson("{\"string\":\"---\"}")
    expected := "  ---\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Standalone interpolation should not alter surrounding whitespace.
func TestInterpolationAmpersandStandalone(t *testing.T) { 
    template := "  {{&string}}\n"
    data     := mustDecodeJson("{\"string\":\"---\"}")
    expected := "  ---\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Superfluous in-tag whitespace should be ignored.
func TestInterpolationInterpolationWithPadding(t *testing.T) { 
    template := "|{{ string }}|"
    data     := mustDecodeJson("{\"string\":\"---\"}")
    expected := "|---|"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Superfluous in-tag whitespace should be ignored.
func TestInterpolationTripleMustacheWithPadding(t *testing.T) { 
    template := "|{{{ string }}}|"
    data     := mustDecodeJson("{\"string\":\"---\"}")
    expected := "|---|"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Superfluous in-tag whitespace should be ignored.
func TestInterpolationAmpersandWithPadding(t *testing.T) { 
    template := "|{{& string }}|"
    data     := mustDecodeJson("{\"string\":\"---\"}")
    expected := "|---|"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}




// -----------------------------------------------------------------------------

// Falsey sections should have their contents rendered.
func TestInvertedFalsey(t *testing.T) { 
    template := "\"{{^boolean}}This should be rendered.{{/boolean}}\""
    data     := mustDecodeJson("{\"boolean\":false}")
    expected := "\"This should be rendered.\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Truthy sections should have their contents omitted.
func TestInvertedTruthy(t *testing.T) { 
    template := "\"{{^boolean}}This should not be rendered.{{/boolean}}\""
    data     := mustDecodeJson("{\"boolean\":true}")
    expected := "\"\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Objects and hashes should behave like truthy values.
func TestInvertedContext(t *testing.T) { 
    template := "\"{{^context}}Hi {{name}}.{{/context}}\""
    data     := mustDecodeJson("{\"context\":{\"name\":\"Joe\"}}")
    expected := "\"\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Lists should behave like truthy values.
func TestInvertedList(t *testing.T) { 
    template := "\"{{^list}}{{n}}{{/list}}\""
    data     := mustDecodeJson("{\"list\":[{\"n\":1},{\"n\":2},{\"n\":3}]}")
    expected := "\"\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Empty lists should behave like falsey values.
func TestInvertedEmptyList(t *testing.T) { 
    template := "\"{{^list}}Yay lists!{{/list}}\""
    data     := mustDecodeJson("{\"list\":[]}")
    expected := "\"Yay lists!\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Multiple inverted sections per template should be permitted.
func TestInvertedDoubled(t *testing.T) { 
    template := "{{^bool}}\n* first\n{{/bool}}\n* {{two}}\n{{^bool}}\n* third\n{{/bool}}\n"
    data     := mustDecodeJson("{\"bool\":false,\"two\":\"second\"}")
    expected := "* first\n* second\n* third\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Nested falsey sections should have their contents rendered.
func TestInvertedNestedFalsey(t *testing.T) { 
    template := "| A {{^bool}}B {{^bool}}C{{/bool}} D{{/bool}} E |"
    data     := mustDecodeJson("{\"bool\":false}")
    expected := "| A B C D E |"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Nested truthy sections should be omitted.
func TestInvertedNestedTruthy(t *testing.T) { 
    template := "| A {{^bool}}B {{^bool}}C{{/bool}} D{{/bool}} E |"
    data     := mustDecodeJson("{\"bool\":true}")
    expected := "| A  E |"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Failed context lookups should be considered falsey.
func TestInvertedContextMisses(t *testing.T) { 
    template := "[{{^missing}}Cannot find key 'missing'!{{/missing}}]"
    expected := "[Cannot find key 'missing'!]"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Dotted names should be valid for Inverted Section tags.
func TestInvertedDottedNamesTruthy(t *testing.T) { 
    template := "\"{{^a.b.c}}Not Here{{/a.b.c}}\" == \"\""
    data     := mustDecodeJson("{\"a\":{\"b\":{\"c\":true}}}")
    expected := "\"\" == \"\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Dotted names should be valid for Inverted Section tags.
func TestInvertedDottedNamesFalsey(t *testing.T) { 
    template := "\"{{^a.b.c}}Not Here{{/a.b.c}}\" == \"Not Here\""
    data     := mustDecodeJson("{\"a\":{\"b\":{\"c\":false}}}")
    expected := "\"Not Here\" == \"Not Here\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Dotted names that cannot be resolved should be considered falsey.
func TestInvertedDottedNamesBrokenChains(t *testing.T) { 
    template := "\"{{^a.b.c}}Not Here{{/a.b.c}}\" == \"Not Here\""
    data     := mustDecodeJson("{\"a\":{}}")
    expected := "\"Not Here\" == \"Not Here\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Inverted sections should not alter surrounding whitespace.
func TestInvertedSurroundingWhitespace(t *testing.T) { 
    template := " | {{^boolean}}\t|\t{{/boolean}} | \n"
    data     := mustDecodeJson("{\"boolean\":false}")
    expected := " | \t|\t | \n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Inverted should not alter internal whitespace.
func TestInvertedInternalWhitespace(t *testing.T) { 
    template := " | {{^boolean}} {{! Important Whitespace }}\n {{/boolean}} | \n"
    data     := mustDecodeJson("{\"boolean\":false}")
    expected := " |  \n  | \n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Single-line sections should not alter surrounding whitespace.
func TestInvertedIndentedInlineSections(t *testing.T) { 
    template := " {{^boolean}}NO{{/boolean}}\n {{^boolean}}WAY{{/boolean}}\n"
    data     := mustDecodeJson("{\"boolean\":false}")
    expected := " NO\n WAY\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Standalone lines should be removed from the template.
func TestInvertedStandaloneLines(t *testing.T) { 
    template := "| This Is\n{{^boolean}}\n|\n{{/boolean}}\n| A Line\n"
    data     := mustDecodeJson("{\"boolean\":false}")
    expected := "| This Is\n|\n| A Line\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Standalone indented lines should be removed from the template.
func TestInvertedStandaloneIndentedLines(t *testing.T) { 
    template := "| This Is\n  {{^boolean}}\n|\n  {{/boolean}}\n| A Line\n"
    data     := mustDecodeJson("{\"boolean\":false}")
    expected := "| This Is\n|\n| A Line\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// "\r\n" should be considered a newline for standalone tags.
func TestInvertedStandaloneLineEndings(t *testing.T) { 
    template := "|\r\n{{^boolean}}\r\n{{/boolean}}\r\n|"
    data     := mustDecodeJson("{\"boolean\":false}")
    expected := "|\r\n|"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Standalone tags should not require a newline to precede them.
func TestInvertedStandaloneWithoutPreviousLine(t *testing.T) { 
    template := "  {{^boolean}}\n^{{/boolean}}\n/"
    data     := mustDecodeJson("{\"boolean\":false}")
    expected := "^\n/"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Standalone tags should not require a newline to follow them.
func TestInvertedStandaloneWithoutNewline(t *testing.T) { 
    template := "^{{^boolean}}\n/\n  {{/boolean}}"
    data     := mustDecodeJson("{\"boolean\":false}")
    expected := "^\n/\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Superfluous in-tag whitespace should be ignored.
func TestInvertedPadding(t *testing.T) { 
    template := "|{{^ boolean }}={{/ boolean }}|"
    data     := mustDecodeJson("{\"boolean\":false}")
    expected := "|=|"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}




// -----------------------------------------------------------------------------

// The greater-than operator should expand to the named partial.
func TestPartialsBasicBehavior(t *testing.T) { 
    ioutil.WriteFile("text", []byte("from partial"), 0666)
    defer os.Remove("text")
    
    template := "\"{{>text}}\""
    expected := "\"from partial\""
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// The empty string should be used when the named partial is not found.
func TestPartialsFailedLookup(t *testing.T) { 
    template := "\"{{>text}}\""
    expected := "\"\""
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// The greater-than operator should operate within the current context.
func TestPartialsContext(t *testing.T) { 
    ioutil.WriteFile("partial", []byte("*{{text}}*"), 0666)
    defer os.Remove("partial")
    
    template := "\"{{>partial}}\""
    data     := mustDecodeJson("{\"text\":\"content\"}")
    expected := "\"*content*\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// The greater-than operator should properly recurse.
func TestPartialsRecursion(t *testing.T) { 
    ioutil.WriteFile("node", []byte("{{content}}<{{#nodes}}{{>node}}{{/nodes}}>"), 0666)
    defer os.Remove("node")
    
    template := "{{>node}}"
    data     := mustDecodeJson("{\"content\":\"X\",\"nodes\":[{\"content\":\"Y\",\"nodes\":[]}]}")
    expected := "X<Y<>>"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// The greater-than operator should not alter surrounding whitespace.
func TestPartialsSurroundingWhitespace(t *testing.T) { 
    ioutil.WriteFile("partial", []byte("\t|\t"), 0666)
    defer os.Remove("partial")
    
    template := "| {{>partial}} |"
    expected := "| \t|\t |"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Whitespace should be left untouched.
func TestPartialsInlineIndentation(t *testing.T) { 
    ioutil.WriteFile("partial", []byte(">\n>"), 0666)
    defer os.Remove("partial")
    
    template := "  {{data}}  {{> partial}}\n"
    data     := mustDecodeJson("{\"data\":\"|\"}")
    expected := "  |  >\n>\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// "\r\n" should be considered a newline for standalone tags.
func TestPartialsStandaloneLineEndings(t *testing.T) { 
    ioutil.WriteFile("partial", []byte(">"), 0666)
    defer os.Remove("partial")
    
    template := "|\r\n{{>partial}}\r\n|"
    expected := "|\r\n>|"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Standalone tags should not require a newline to precede them.
func TestPartialsStandaloneWithoutPreviousLine(t *testing.T) { 
    ioutil.WriteFile("partial", []byte(">\n>"), 0666)
    defer os.Remove("partial")
    
    template := "  {{>partial}}\n>"
    expected := "  >\n  >>"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Standalone tags should not require a newline to follow them.
func TestPartialsStandaloneWithoutNewline(t *testing.T) { 
    ioutil.WriteFile("partial", []byte(">\n>"), 0666)
    defer os.Remove("partial")
    
    template := ">\n  {{>partial}}"
    expected := ">\n  >\n  >"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Each line of the partial should be indented before rendering.
func TestPartialsStandaloneIndentation(t *testing.T) { 
    ioutil.WriteFile("partial", []byte("|\n{{{content}}}\n|\n"), 0666)
    defer os.Remove("partial")
    
    template := "\\\n {{>partial}}\n/\n"
    data     := mustDecodeJson("{\"content\":\"\\u003c\\n-\\u003e\"}")
    expected := "\\\n |\n <\n->\n |\n/\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Superfluous in-tag whitespace should be ignored.
func TestPartialsPaddingWhitespace(t *testing.T) { 
    ioutil.WriteFile("partial", []byte("[]"), 0666)
    defer os.Remove("partial")
    
    template := "|{{> partial }}|"
    data     := mustDecodeJson("{\"boolean\":true}")
    expected := "|[]|"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}




// -----------------------------------------------------------------------------

// Truthy sections should have their contents rendered.
func TestSectionsTruthy(t *testing.T) { 
    template := "\"{{#boolean}}This should be rendered.{{/boolean}}\""
    data     := mustDecodeJson("{\"boolean\":true}")
    expected := "\"This should be rendered.\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Falsey sections should have their contents omitted.
func TestSectionsFalsey(t *testing.T) { 
    template := "\"{{#boolean}}This should not be rendered.{{/boolean}}\""
    data     := mustDecodeJson("{\"boolean\":false}")
    expected := "\"\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Objects and hashes should be pushed onto the context stack.
func TestSectionsContext(t *testing.T) { 
    template := "\"{{#context}}Hi {{name}}.{{/context}}\""
    data     := mustDecodeJson("{\"context\":{\"name\":\"Joe\"}}")
    expected := "\"Hi Joe.\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// All elements on the context stack should be accessible.
func TestSectionsDeeplyNestedContexts(t *testing.T) { 
    template := "{{#a}}\n{{one}}\n{{#b}}\n{{one}}{{two}}{{one}}\n{{#c}}\n{{one}}{{two}}{{three}}{{two}}{{one}}\n{{#d}}\n{{one}}{{two}}{{three}}{{four}}{{three}}{{two}}{{one}}\n{{#e}}\n{{one}}{{two}}{{three}}{{four}}{{five}}{{four}}{{three}}{{two}}{{one}}\n{{/e}}\n{{one}}{{two}}{{three}}{{four}}{{three}}{{two}}{{one}}\n{{/d}}\n{{one}}{{two}}{{three}}{{two}}{{one}}\n{{/c}}\n{{one}}{{two}}{{one}}\n{{/b}}\n{{one}}\n{{/a}}\n"
    data     := mustDecodeJson("{\"a\":{\"one\":1},\"b\":{\"two\":2},\"c\":{\"three\":3},\"d\":{\"four\":4},\"e\":{\"five\":5}}")
    expected := "1\n121\n12321\n1234321\n123454321\n1234321\n12321\n121\n1\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Lists should be iterated; list items should visit the context stack.
func TestSectionsList(t *testing.T) { 
    template := "\"{{#list}}{{item}}{{/list}}\""
    data     := mustDecodeJson("{\"list\":[{\"item\":1},{\"item\":2},{\"item\":3}]}")
    expected := "\"123\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Empty lists should behave like falsey values.
func TestSectionsEmptyList(t *testing.T) { 
    template := "\"{{#list}}Yay lists!{{/list}}\""
    data     := mustDecodeJson("{\"list\":[]}")
    expected := "\"\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Multiple sections per template should be permitted.
func TestSectionsDoubled(t *testing.T) { 
    template := "{{#bool}}\n* first\n{{/bool}}\n* {{two}}\n{{#bool}}\n* third\n{{/bool}}\n"
    data     := mustDecodeJson("{\"bool\":true,\"two\":\"second\"}")
    expected := "* first\n* second\n* third\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Nested truthy sections should have their contents rendered.
func TestSectionsNestedTruthy(t *testing.T) { 
    template := "| A {{#bool}}B {{#bool}}C{{/bool}} D{{/bool}} E |"
    data     := mustDecodeJson("{\"bool\":true}")
    expected := "| A B C D E |"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Nested falsey sections should be omitted.
func TestSectionsNestedFalsey(t *testing.T) { 
    template := "| A {{#bool}}B {{#bool}}C{{/bool}} D{{/bool}} E |"
    data     := mustDecodeJson("{\"bool\":false}")
    expected := "| A  E |"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Failed context lookups should be considered falsey.
func TestSectionsContextMisses(t *testing.T) { 
    template := "[{{#missing}}Found key 'missing'!{{/missing}}]"
    expected := "[]"
    actual   := Render(template)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Implicit iterators should directly interpolate strings.
func TestSectionsImplicitIteratorString(t *testing.T) { 
    template := "\"{{#list}}({{.}}){{/list}}\""
    data     := mustDecodeJson("{\"list\":[\"a\",\"b\",\"c\",\"d\",\"e\"]}")
    expected := "\"(a)(b)(c)(d)(e)\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Implicit iterators should cast integers to strings and interpolate.
func TestSectionsImplicitIteratorInteger(t *testing.T) { 
    template := "\"{{#list}}({{.}}){{/list}}\""
    data     := mustDecodeJson("{\"list\":[1,2,3,4,5]}")
    expected := "\"(1)(2)(3)(4)(5)\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Implicit iterators should cast decimals to strings and interpolate.
func TestSectionsImplicitIteratorDecimal(t *testing.T) { 
    template := "\"{{#list}}({{.}}){{/list}}\""
    data     := mustDecodeJson("{\"list\":[1.1,2.2,3.3,4.4,5.5]}")
    expected := "\"(1.1)(2.2)(3.3)(4.4)(5.5)\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Dotted names should be valid for Section tags.
func TestSectionsDottedNamesTruthy(t *testing.T) { 
    template := "\"{{#a.b.c}}Here{{/a.b.c}}\" == \"Here\""
    data     := mustDecodeJson("{\"a\":{\"b\":{\"c\":true}}}")
    expected := "\"Here\" == \"Here\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Dotted names should be valid for Section tags.
func TestSectionsDottedNamesFalsey(t *testing.T) { 
    template := "\"{{#a.b.c}}Here{{/a.b.c}}\" == \"\""
    data     := mustDecodeJson("{\"a\":{\"b\":{\"c\":false}}}")
    expected := "\"\" == \"\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Dotted names that cannot be resolved should be considered falsey.
func TestSectionsDottedNamesBrokenChains(t *testing.T) { 
    template := "\"{{#a.b.c}}Here{{/a.b.c}}\" == \"\""
    data     := mustDecodeJson("{\"a\":{}}")
    expected := "\"\" == \"\""
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Sections should not alter surrounding whitespace.
func TestSectionsSurroundingWhitespace(t *testing.T) { 
    template := " | {{#boolean}}\t|\t{{/boolean}} | \n"
    data     := mustDecodeJson("{\"boolean\":true}")
    expected := " | \t|\t | \n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Sections should not alter internal whitespace.
func TestSectionsInternalWhitespace(t *testing.T) { 
    template := " | {{#boolean}} {{! Important Whitespace }}\n {{/boolean}} | \n"
    data     := mustDecodeJson("{\"boolean\":true}")
    expected := " |  \n  | \n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Single-line sections should not alter surrounding whitespace.
func TestSectionsIndentedInlineSections(t *testing.T) { 
    template := " {{#boolean}}YES{{/boolean}}\n {{#boolean}}GOOD{{/boolean}}\n"
    data     := mustDecodeJson("{\"boolean\":true}")
    expected := " YES\n GOOD\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Standalone lines should be removed from the template.
func TestSectionsStandaloneLines(t *testing.T) { 
    template := "| This Is\n{{#boolean}}\n|\n{{/boolean}}\n| A Line\n"
    data     := mustDecodeJson("{\"boolean\":true}")
    expected := "| This Is\n|\n| A Line\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Indented standalone lines should be removed from the template.
func TestSectionsIndentedStandaloneLines(t *testing.T) { 
    template := "| This Is\n  {{#boolean}}\n|\n  {{/boolean}}\n| A Line\n"
    data     := mustDecodeJson("{\"boolean\":true}")
    expected := "| This Is\n|\n| A Line\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// "\r\n" should be considered a newline for standalone tags.
func TestSectionsStandaloneLineEndings(t *testing.T) { 
    template := "|\r\n{{#boolean}}\r\n{{/boolean}}\r\n|"
    data     := mustDecodeJson("{\"boolean\":true}")
    expected := "|\r\n|"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Standalone tags should not require a newline to precede them.
func TestSectionsStandaloneWithoutPreviousLine(t *testing.T) { 
    template := "  {{#boolean}}\n#{{/boolean}}\n/"
    data     := mustDecodeJson("{\"boolean\":true}")
    expected := "#\n/"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Standalone tags should not require a newline to follow them.
func TestSectionsStandaloneWithoutNewline(t *testing.T) { 
    template := "#{{#boolean}}\n/\n  {{/boolean}}"
    data     := mustDecodeJson("{\"boolean\":true}")
    expected := "#\n/\n"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}

// Superfluous in-tag whitespace should be ignored.
func TestSectionsPadding(t *testing.T) { 
    template := "|{{# boolean }}={{/ boolean }}|"
    data     := mustDecodeJson("{\"boolean\":true}")
    expected := "|=|"
    actual   := Render(template, data)

    if actual != expected {
        t.Errorf("returned %#v, expected %#v", actual, expected)
    }
}
