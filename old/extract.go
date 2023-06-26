package main

type ScannerSQL interface {
	Scan() Token
	Take(int, int) []byte
	TakeString(int, int) string
}

func iterSQL(r io.Reader, keywords KeywordSet) error {
	var (
		scan ScannerSQL
		err  error
	)
	scan, err = Scan(r, keywords)
	if err != nil {
		return err
	}
	return iterStatement(scan)
}

func ensureEOL(scan ScannerSQL, offset int) (string, error) {
	tok := scan.Scan()
	if tok.Type != EOL && tok.Type != EOF {
		return "", fmt.Errorf("expected ',' or end of file but got %s", tok)
	}
	return scan.TakeString(offset, tok.Offset+tok.Length()), nil
}

func iterStatement(scan ScannerSQL) error {
	for {
		tok := scan.Scan()
		switch tok.Type {
		case Comment:
			continue
		case EOF:
			return nil
		case Keyword:
		default:
			return fmt.Errorf("unexpected token %s (%d)", tok, tok.Offset)
		}
		if err := startStatement(scan, tok, true); err != nil {
			return err
		}
	}
	return nil
}

func startStatement(scan ScannerSQL, tok Token, top bool) error {
	var (
		str string
		err error
	)
	switch tok.Literal {
	case "BEGIN":
		str, err = blockStatement(scan, tok, "END")
	case "CASE":
		str, err = caseStatement(scan, tok)
	case "IF":
		str, err = ifStatement(scan, tok)
	case "WHILE":
		str, err = whileStatement(scan, tok)
	case "CREATE PROCEDURE", "CREATE OR REPLACE PROCEDURE":
		str, err = procStatement(scan, tok)
	default:
		str, err = basicStatement(scan, tok)
	}
	if err == nil && top {
		fmt.Println(str)
	}
	return err
}

func caseStatement(scan ScannerSQL, tok Token) (string, error) {
	curr, err := skipUntilKeyword(scan, "WHEN", "ELSE", "END")
	if err != nil {
		return "", err
	}
	for curr.Literal != "END" {
		if curr.Type == Keyword && curr.Literal == "WHEN" {
			curr, err = skipUntilKeyword(scan, "THEN")
			if err != nil {
				return "", err
			}
		}
		for {
			curr = scan.Scan()
			if curr.Type == Keyword {
				if curr.Literal == "WHEN" || curr.Literal == "ELSE" || curr.Literal == "END" {
					break
				}
			}
		}
	}
	return ensureEOL(scan, tok.Offset)
}

func procStatement(scan ScannerSQL, tok Token) (string, error) {
	_, err := skipUntilKeyword(scan, "BEGIN")
	if err != nil {
		return "", err
	}
	return blockStatement(scan, tok, "END")
}

func ifStatement(scan ScannerSQL, tok Token) (string, error) {
	curr, err := skipUntilKeyword(scan, "THEN")
	if err != nil {
		return "", err
	}
	for {
		curr = scan.Scan()
		if curr.Type == Keyword {
			if curr.Literal == "END IF" {
				break
			} else if curr.Literal == "ELSEIF" || curr.Literal == "ELSE IF" {
				return ifStatement(scan, tok)
			} else if curr.Literal == "ELSE" {
				return blockStatement(scan, tok, "END IF")
			}
		}
		if err := startStatement(scan, curr, false); err != nil {
			return "", err
		}
	}
	return ensureEOL(scan, tok.Offset)
}

func whileStatement(scan ScannerSQL, tok Token) (string, error) {
	_, err := skipUntilKeyword(scan, "DO")
	if err != nil {
		return "", err
	}
	return blockStatement(scan, tok, "END WHILE")
}

func blockStatement(scan ScannerSQL, tok Token, end string) (string, error) {
	var curr Token
	for {
		curr = scan.Scan()
		if curr.Type == Keyword && curr.Literal == end {
			break
		}
		if err := startStatement(scan, curr, false); err != nil {
			return "", err
		}
	}
	return ensureEOL(scan, tok.Offset)
}

func basicStatement(scan ScannerSQL, tok Token) (string, error) {
	var curr Token
	for {
		curr = scan.Scan()
		if curr.Type == EOL {
			break
		}
		if curr.Type == EOF {
			return "", fmt.Errorf("unexpected end of statement - eof found")
		}
	}
	return scan.TakeString(tok.Offset, curr.Offset+curr.Length()), nil
}

func skipUntilKeyword(scan ScannerSQL, kw ...string) (Token, error) {
	sort.Strings(kw)
	for curr := scan.Scan(); curr.Type != EOF; curr = scan.Scan() {
		if curr.Type != Keyword {
			continue
		}
		i := sort.SearchStrings(kw, curr.Literal)
		if i < len(kw) && kw[i] == curr.Literal {
			return curr, nil
		}
	}
	return Token{}, fmt.Errorf("EOF file reached without %q", kw)
}
