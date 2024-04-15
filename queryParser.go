package hypeql

import (
	"fmt"
	"strconv"
	"strings"
)

// Converts the request body content to an interfaces slice to process it in "Process" function
func RequestBodyParse(body string) ([]interface{}, error) {
	body = strings.Trim(strings.ReplaceAll(body, "\t", ""), " ")
	res := []interface{}{}
	varOpened := false
	keyWrite := true
	vars := map[string]interface{}{}
	key := ""
	quoteOpened := false
	valueSlash := false
	comment := false
	value := ""
	varIsString := false
	k := ""

	for _, c := range body {
		if c != '\\' && c != '"' && c != 'n' && varOpened && !keyWrite && valueSlash {
			valueSlash = false
		}

		if comment && c != '\n' {
			continue
		}

		switch c {

		case '{':
			if cleanUp(k) != "" {
				k += string(c)
			}

		case '\n':
			if varOpened {
				return []interface{}{}, fmt.Errorf("argument declaration have interputted by \\n symbol")
			}

			cu := cleanUp(k)
			if cu != "" {
				if strings.Contains(cu, "{") {
					k += string(c)
				} else {
					res = append(res, cu)
					k = ""
				}
			}
			comment = false

		case '#':
			if !varOpened {
				comment = true
			}

		case '(':
			if varOpened {
				if !keyWrite {
					value += string(c)
				}
			} else {
				if !strings.Contains(k, "{") {
					varIsString = false
					varOpened = true
					key = ""
					value = ""
					vars = map[string]interface{}{}
					keyWrite = true
					quoteOpened = false
				}
			}

		case ':':
			if varOpened {
				if keyWrite {
					keyWrite = false
				} else {
					value += string(c)
				}
			}

		case ' ':
			if varOpened {
				if !keyWrite {
					value += string(c)
				}
			} else {
				cu := cleanUp(k)
				if cu != "" {
					if strings.Contains(cu, "{") {
						k += string(c)
					}
				}
			}

		case '\\':
			if varOpened {
				if !keyWrite {
					if valueSlash {
						value += string(c)
						valueSlash = false
					} else {
						valueSlash = true
					}
				}
			}

		case '"':
			if varOpened {
				if !keyWrite {
					if valueSlash {
						value += string(c)
						valueSlash = false
					} else if quoteOpened {
						quoteOpened = false
					} else {
						varIsString = true
						quoteOpened = true
					}
				}
			}

		case ',':
			if varOpened {
				if !keyWrite {
					if !quoteOpened {
						if cleanUp(key) != "" {
							if len(value) > 0 && value[0] == ' ' {
								value = value[1:]
							}

							vars[key] = func(a string) any {
								if varIsString {
									return a
								}

								if i, err := strconv.Atoi(a); err == nil {
									return i
								}

								return a
							}(value)

							varIsString = false
							key = ""
							value = ""
							keyWrite = true
						}
					} else {
						value += string(c)
					}
				}
			} else {
				cu := cleanUp(k)
				if cu != "" {
					if !strings.Contains(cu, "{") {
						res = append(res, cu)
						k = ""
					} else {
						k += string(c)
					}
				}
			}

		case ')':
			if varOpened {
				if keyWrite {
					varOpened = false
				} else {
					if !quoteOpened {
						if cleanUp(key) != "" {
							if len(value) > 0 && value[0] == ' ' {
								value = value[1:]
							}

							// Two aviables types in the args: int, string
							vars[key] = func(a string) any {
								if varIsString {
									return a
								}

								if i, err := strconv.Atoi(a); err == nil {
									return i
								}

								return a
							}(value)

							varOpened = false
						}
					} else {
						value += string(c)
					}
				}
			}

		case '}':
			if varOpened {
				if !keyWrite {
					value += string(c)
				}
			} else {
				cu := cleanUp(k)
				if cu != "" {
					if !strings.Contains(cu, "{") {
						res = append(res, cu)
						k = ""
					} else if strings.Count(cu, "{") == strings.Count(cu, "}")+1 {
						k += string(c)
						i := strings.Index(k, "{")
						name := cleanUp(k[:i])
						block := k[i:]
						u, err := RequestBodyParse(block)
						if err != nil {
							return []interface{}{}, err
						}

						a := []interface{}{
							name,
							u,
						}

						if len(vars) != 0 {
							a = append(a, vars)
						}

						res = append(res, a)
						k = ""
						vars = map[string]interface{}{}
					} else {
						k += string(c)
					}
				}
			}

		default:
			if varOpened {
				if keyWrite {
					key += string(c)
				} else {
					if c == 'n' && valueSlash {
						value += "\n"
						valueSlash = false
					} else {
						value += string(c)
					}
				}
			} else {
				k += string(c)
			}
		}
	}

	if strings.Contains(k, "{") {
		return []interface{}{}, fmt.Errorf("the curly bracket is not closed")
	} else if varOpened {
		return []interface{}{}, fmt.Errorf("the arguments in parentheses were not written")
	}

	return res, nil
}

// Cleans whitespace symbols.
// Use it for the body parsing, but DON'T use it to clean the full code to process it because space symbols also removing from an argument brackets value
func cleanUp(a string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(a, " ", ""), "\t", ""), "\n", "")
}
