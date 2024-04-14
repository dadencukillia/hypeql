package hypeql

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// Generates json with errors list from argument
func jsonStringWithErrors(errors []string) string {
	cont, _ := json.Marshal(map[string]interface{}{
		"errors": errors,
	})

	return string(cont)
}

// Processes a request's brances recursively
func recursiveProcessRequest(r []interface{}, ctx map[string]interface{}, path []string, ds interface{}) (interface{}, error) {
	// Map for returning
	var ret map[string]interface{} = map[string]interface{}{}
	branchRefVal := reflect.ValueOf(ds)

	// Receiving the "Resolve" method
	if resolveMethod := branchRefVal.MethodByName("Resolve"); resolveMethod.IsValid() {
		neededFields := []string{} // List of fields tags that needed for this recursive cycle

		// Filling neededFields list
		for _, i := range r {
			kind := reflect.TypeOf(i).Kind()
			if st, ok := i.(string); ok {
				neededFields = append(neededFields, st)
			} else if kind == reflect.Slice {
				q := reflect.ValueOf(i)
				if q.Len() > 1 {
					if st, ok := q.Index(0).Interface().(string); ok {
						neededFields = append(neededFields, st)
					}
				}
			}
		}

		// Calling the "Resolve" method that can change context values (you can use context values in another resolver functions)
		// Use the "Resolve" method to connect with a database for example
		// func (a ResponseStruct) Resolve(contextMap *map[string]interface{}, neededFields []string) {...}
		resolveMethod.Call([]reflect.Value{
			reflect.ValueOf(&ctx),
			reflect.ValueOf(neededFields),
		})
	}

	// List of already processed fields (in the case when one fields mentioned many times in the request body)
	checked := []string{}

	// Traversing and receiving values of needed fields by listed tags
a:
	for _, i := range r {
		kind := reflect.TypeOf(i).Kind()

		// Single (Basic) value (i = field's tag)
		if kind == reflect.String {
			// Getting field's tag name
			key := fmt.Sprint(i)

			// Skipping if field already processed
			if slices.Contains(checked, key) {
				continue
			}

			checked = append(checked, key)

			newPath := strings.Join(append(path, key), ".")

			// Finding field by tag
			for i := 0; i < branchRefVal.NumField(); i++ {
				fieldType := branchRefVal.Type().Field(i)

				if fieldType.Tag.Get("json") == key && fieldType.Type.Kind() != reflect.Func {
					// If field found

					// Getting function middleware name
					if funcName := fieldType.Tag.Get("fun"); funcName != "" {
						if q := branchRefVal.MethodByName(funcName); q.IsValid() {

							// Calling middleware function
							// Middleware function can replace value of field and use context values (from argument)
							newVal := q.Call([]reflect.Value{
								reflect.ValueOf(&ctx),
							})

							if len(newVal) > 0 && !newVal[0].IsZero() {
								ret[key] = newVal[0].Interface()

								continue a
							}
						}
					}

					// Use field's value if middleware function is not found
					ret[key] = branchRefVal.Field(i).Interface()

					// Finish field finding process (for loop)
					continue a
				}
			}

			// If field does not found
			return []interface{}{}, fmt.Errorf(newPath + " not found in the struct")
		} else if kind == reflect.Slice { // List of objects (branches) (i = [field's name, object's needed fields, arguments])
			sliceVal, _ := i.([]interface{})
			arguments := map[string]interface{}{}

			if len(sliceVal) != 2 && len(sliceVal) != 3 {
				return []interface{}{}, fmt.Errorf(strings.Join(path, ".") + " length of list must have two or three elements")
			}

			// Slice has arguments values in third element
			if len(sliceVal) == 3 {
				if newArguments, ok := sliceVal[2].(map[string]interface{}); ok {
					arguments = newArguments
				}
			}

			// Getting field's tag name
			tagName, ok := sliceVal[0].(string)
			if !ok {
				return []interface{}{}, fmt.Errorf(strings.Join(path, ".") + " first argument of list must have string type")
			}

			// Skipping if field already processed
			if slices.Contains(checked, tagName) {
				continue
			}
			checked = append(checked, tagName)

			newPath := strings.Join(append(path, tagName), ".")

			// Needed fields of object from field
			neededFields, ok := sliceVal[1].([]interface{})
			if !ok {
				return []interface{}{}, fmt.Errorf(newPath + " second argument of list must have slice type")
			}

			// Finding field by tag
			for i := 0; i < branchRefVal.NumField(); i++ {
				sf := branchRefVal.Type().Field(i)

				if sf.Tag.Get("json") == tagName && sf.Type.Kind() == reflect.Slice {
					// When field found
					l := branchRefVal.Field(i)

					// Getting middleware function's name
					if funcName := sf.Tag.Get("fun"); funcName != "" {
						if q := branchRefVal.MethodByName(funcName); q.IsValid() {

							// Calling middleware function
							// Middleware function can replace value of field and use context values (from argument)
							newVal := q.Call([]reflect.Value{
								reflect.ValueOf(&ctx),
								reflect.ValueOf(arguments), // Arguments of objects list from body
							})

							if len(newVal) > 0 && !newVal[0].IsZero() && newVal[0].Type().Kind() == reflect.Slice {
								l = newVal[0]
							}
						}
					}

					objects := []interface{}{}

					// Parsing objects in a new recursion iteration (new branch)
					for i := 0; i < l.Len(); i++ {
						p := l.Index(i).Interface()

						if reflect.TypeOf(p).Kind() != reflect.Struct {
							continue
						}

						i, err := recursiveProcessRequest(neededFields, ctx, append(path, tagName), p)
						if err != nil {
							return []interface{}{}, err
						}

						objects = append(objects, i)
					}

					// Writing parsed objects
					ret[tagName] = objects

					continue a
				}
			}

			// Field not found
			return []interface{}{}, fmt.Errorf(newPath + " field not found in the struct")
		} else {
			return []interface{}{}, fmt.Errorf(strings.Join(path, ".") + " incorrect data type. The String or Slice types only allowed")
		}
	}

	return ret, nil
}

// Processes a request body and returns a result (the first is JSON string, the second is it error value)
func Process(requestBody []interface{}, dataStruct interface{}, initContext map[string]interface{}) (string, bool) {

	// dataStruct argument must be Struct
	if reflect.TypeOf(dataStruct).Kind() != reflect.Struct {
		log.Panicln("dataStruct argument must contains empty instance of struct")
	}

	// Start recursion to process all fields in the request
	i, err := recursiveProcessRequest(requestBody, initContext, []string{}, dataStruct)
	if err != nil {
		return jsonStringWithErrors([]string{err.Error()}), true
	}

	// Converting result to JSON string and return
	q, err := json.Marshal(i)
	if err != nil {
		return jsonStringWithErrors([]string{"JSON converting error"}), true
	}

	return string(q), false
}

// Cleans whitespace symbols.
// Use it for the body parsing, but DON'T use it to clean the full code to process it because space symbols also removing from an argument brackets value
func cleanUp(a string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(a, " ", ""), "\t", ""), "\n", "")
}

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
					} else {
						res = append(res, cu)
						k = ""
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
							if i, err := strconv.Atoi(value); err == nil {
								vars[key] = i
							} else {
								vars[key] = value
							}

							key = ""
							value = ""
							keyWrite = true
						}
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
							if i, err := strconv.Atoi(value); err == nil {
								vars[key] = i
							} else {
								vars[key] = value
							}

							varOpened = false
						}
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
