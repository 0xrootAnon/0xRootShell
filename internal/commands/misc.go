// 0xRootShell — A minimalist, aesthetic terminal for creators
// Copyright (c) 2025 Khwahish Sharma (aka 0xRootAnon)
//
// Licensed under the GNU General Public License v3.0 or later (GPLv3+).
// You may obtain a copy of the License at
// https://www.gnu.org/licenses/gpl-3.0.html
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.

package commands

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

func CmdCalc(args []string) string {
	if len(args) == 0 {
		return "calc: expected expression, e.g. `calc 2+2*3`"
	}
	expr := strings.Join(args, " ")
	val, err := evalSimple(expr)
	if err != nil {
		return "calc error: " + err.Error()
	}
	return fmt.Sprintf("%s = %g", expr, val)
}

func evalSimple(s string) (float64, error) {
	type tok struct {
		typ string
		val float64
		op  rune
	}
	var toks []tok
	i := 0
	for i < len(s) {
		c := rune(s[i])
		if unicode.IsSpace(c) {
			i++
			continue
		}
		if unicode.IsDigit(c) || c == '.' {
			j := i
			for j < len(s) && (unicode.IsDigit(rune(s[j])) || rune(s[j]) == '.') {
				j++
			}
			numStr := s[i:j]
			f, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, err
			}
			toks = append(toks, tok{typ: "num", val: f})
			i = j
			continue
		}
		if strings.ContainsRune("+-*/()", c) {
			if c == '(' || c == ')' {
				toks = append(toks, tok{typ: string(c)})
			} else {
				toks = append(toks, tok{typ: "op", op: c})
			}
			i++
			continue
		}
		return 0, fmt.Errorf("invalid char: %c", c)
	}
	outQ := []tok{}
	opS := []tok{}
	prec := func(r rune) int {
		switch r {
		case '+', '-':
			return 1
		case '*', '/':
			return 2
		}
		return 0
	}
	for _, t := range toks {
		if t.typ == "num" {
			outQ = append(outQ, t)
			continue
		}
		if t.typ == "op" {
			for len(opS) > 0 {
				top := opS[len(opS)-1]
				if top.typ == "op" && (prec(top.op) >= prec(t.op)) {
					outQ = append(outQ, top)
					opS = opS[:len(opS)-1]
					continue
				}
				break
			}
			opS = append(opS, t)
			continue
		}
		if t.typ == "(" {
			opS = append(opS, t)
			continue
		}
		if t.typ == ")" {
			found := false
			for len(opS) > 0 {
				top := opS[len(opS)-1]
				opS = opS[:len(opS)-1]
				if top.typ == "(" {
					found = true
					break
				}
				outQ = append(outQ, top)
			}
			if !found {
				return 0, fmt.Errorf("mismatched parentheses")
			}
		}
	}
	for len(opS) > 0 {
		top := opS[len(opS)-1]
		opS = opS[:len(opS)-1]
		if top.typ == "(" || top.typ == ")" {
			return 0, fmt.Errorf("mismatched parentheses")
		}
		outQ = append(outQ, top)
	}
	st := []float64{}
	for _, t := range outQ {
		if t.typ == "num" {
			st = append(st, t.val)
			continue
		}
		if t.typ == "op" {
			if len(st) < 2 {
				return 0, fmt.Errorf("invalid expression")
			}
			b := st[len(st)-1]
			a := st[len(st)-2]
			st = st[:len(st)-2]
			var r float64
			switch t.op {
			case '+':
				r = a + b
			case '-':
				r = a - b
			case '*':
				r = a * b
			case '/':
				r = a / b
			}
			st = append(st, r)
		}
	}
	if len(st) != 1 {
		return 0, fmt.Errorf("invalid expression")
	}
	return st[0], nil
}
