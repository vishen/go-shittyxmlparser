package main

import (
	"fmt"
	"io/ioutil"
	"strings"
)

const (
	START         = "<"
	END           = ">"
	FINISH_NODE   = "</"
	COMMENT_START = "<!--"
	COMMENT_END   = "-->"
)

const (
	START_NODE = iota
	END_NODE
	ATTRIBUTE_NODE
	VALUE_NODE
	COMMENT_NODE
	ERROR
)

type Token struct {
	pos        int
	key        string
	value      string
	token_type int
}

func (t Token) String() string {
	var token_type string
	m := 0

	switch t.token_type {
	case 0:
		token_type = "StartNode"
	case 1:
		token_type = "EndNode"
	case 2:
		token_type = "Attribute"
		m = 1
	case 3:
		token_type = "Value"
	case 4:
		token_type = "CommentNode"
	case 5:
		token_type = "Error"
	}

	switch m {
	case 0:
		return fmt.Sprintf("%s - %s [%d]", token_type, t.value, t.pos)
	case 1:
		return fmt.Sprintf("%s - %s=%s [%d]", token_type, t.key,
			t.value, t.pos)
	}

	return "TokenErrored"
}

type Parser struct {
	source        *string
	source_length int
	cpos          int // Current position.

	tokens []Token

	ignore_next_token bool
}

func GetParser(source *string) *Parser {
	source_length := len(*source)
	return &Parser{source: source,
		source_length: source_length,
		tokens:        make([]Token, 0),
		cpos:          0}
}

func (p *Parser) Tokenize() {

	for {
		if p.cpos >= p.source_length {
			return
		}

		if (p.cpos+2 <= p.source_length) && ((*p.source)[p.cpos:p.cpos+2] == FINISH_NODE) {
			p.cpos += 2
			p.getEndToken()

		} else if (*p.source)[p.cpos:p.cpos+1] == START {
			p.cpos += 1
			p.getStartToken()
			p.getAttributeTokens()

		} else {
			// We have found the text inside a node.
			p.getValueToken()
		}

	}
}

func (p *Parser) isIgnoreToken(s string) {
	if s == "script" || s == "style" {
		p.ignore_next_token = true
	} else {
		p.ignore_next_token = false
	}
}

func (p *Parser) getStartToken() {

	start_token := Token{pos: p.cpos, token_type: START_NODE}
	started_pos := p.cpos
	var value string
	for {
		if p.cpos >= p.source_length {
			p.tokens = append(p.tokens, Token{token_type: ERROR, pos: p.cpos,
				value: "SyntaxError"})
			return
		}

		value = (*p.source)[p.cpos : p.cpos+1]
		if value == " " || value == ">" {
			start_token.value = (*p.source)[started_pos:p.cpos]
			p.isIgnoreToken(start_token.value)
			p.tokens = append(p.tokens, start_token)
			return
		}

		p.cpos += 1

	}
}

func (p *Parser) getEndToken() {

	token := Token{pos: p.cpos, token_type: END_NODE}
	started_pos := p.cpos
	for {
		if p.cpos >= p.source_length {
			p.tokens = append(p.tokens, Token{token_type: ERROR, pos: p.cpos,
				value: "SyntaxError"})
			return
		}

		if (*p.source)[p.cpos:p.cpos+1] == ">" {
			token.value = (*p.source)[started_pos:p.cpos]
			p.isIgnoreToken(token.value)
			p.tokens = append(p.tokens, token)
			p.cpos += 1
			return
		}

		p.cpos += 1

	}
}

func (p *Parser) getAttributeTokens() {
	token := Token{pos: p.cpos, token_type: ATTRIBUTE_NODE}
	start_pos := p.cpos

	in_string := false

	add_token := func() {

		value := strings.TrimSpace((*p.source)[start_pos:p.cpos])
		_add_token := false
		if token.key != "" {
			token.value = value
			_add_token = true

		} else if value != "" {
			token.key = strings.TrimSpace((*p.source)[start_pos:p.cpos])
			_add_token = true

		}

		if _add_token {
			p.tokens = append(p.tokens, token)
			token = Token{pos: p.cpos, token_type: ATTRIBUTE_NODE}
		}
	}

	for {
		if p.cpos >= p.source_length {
			p.tokens = append(p.tokens, Token{token_type: ERROR, pos: p.cpos,
				value: "SyntaxError"})
			return
		}

		switch (*p.source)[p.cpos : p.cpos+1] {
		case ">":
			add_token()
			return
		case " ":
			add_token()
			start_pos = p.cpos
		case "=":
			if !in_string {
				token.key = strings.TrimSpace((*p.source)[start_pos:p.cpos])
				start_pos = p.cpos + 1
			}
		case "'", "\"":
			if in_string {
				in_string = false
			} else {
				in_string = true
			}
		default:
		}

		p.cpos += 1
	}
}

func (p *Parser) getValueToken() {

	token := Token{pos: p.cpos, token_type: VALUE_NODE}
	started_pos := p.cpos
	for {
		if p.cpos >= p.source_length {
			p.tokens = append(p.tokens, Token{token_type: ERROR, pos: p.cpos,
				value: "SyntaxError"})
			return
		}

		switch (*p.source)[p.cpos : p.cpos+1] {
		case "<":

			if !p.ignore_next_token {
				if !p.getCommentToken(started_pos) {
					token.value = strings.TrimSpace((*p.source)[started_pos:p.cpos])

					if token.value != "" {
						p.tokens = append(p.tokens, token)
					}
				}
				return
			}
		case ">":
			started_pos = p.cpos + 1
		}

		p.cpos += 1

	}
}

func (p *Parser) getCommentToken(backtrack_pos int) bool {
	if p.cpos+4 > p.source_length {
		return false
	}

	comment_start_count := len(COMMENT_START)
	comment_end_count := len(COMMENT_END)

	prev_started_pos := p.cpos

	if (*p.source)[p.cpos:p.cpos+comment_start_count] == COMMENT_START {
		p.cpos += comment_start_count
		started_pos := p.cpos
		for {
			if p.cpos+comment_end_count > p.source_length {
				p.cpos = started_pos
				return false
			}

			if (*p.source)[p.cpos:p.cpos+comment_end_count] == COMMENT_END {
				value := strings.TrimSpace((*p.source)[backtrack_pos:prev_started_pos])
				if value != "" {
					value_token := Token{token_type: VALUE_NODE, pos: backtrack_pos,
						value: value}
					comment_token := Token{token_type: COMMENT_NODE,
						pos: started_pos, value: strings.TrimSpace((*p.source)[started_pos:p.cpos])}
					p.tokens = append(p.tokens, value_token, comment_token)
				} else {
					comment_token := Token{token_type: COMMENT_NODE,
						pos:   started_pos,
						value: strings.TrimSpace((*p.source)[started_pos:p.cpos])}
					p.tokens = append(p.tokens, comment_token)
				}

				p.cpos += 3
				return true

			}

			p.cpos += 1

		}
	}

	return false
}

func (p *Parser) cleanTokens() {
	for i, _ := range p.tokens {
		p.tokens[i].value = strings.TrimSpace(p.tokens[i].value)
		p.tokens[i].key = strings.TrimSpace(p.tokens[i].key)
	}

}

func (p *Parser) PrintTokens() {
	for _, token := range p.tokens {
		fmt.Println(token)
	}
}

func main() {
	// source := `<xml attr="sucks">Hello<span> Jonathan </span></xml>`
	filename := "test3.html"
	source_bytes, err := ioutil.ReadFile(filename)

	if err != nil {
		panic(err)
	}

	source := string(source_bytes)

	parser := GetParser(&source)
	parser.Tokenize()
	parser.PrintTokens()

}
