package hypeql

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

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

// Processes a request body and returns a result (the first is JSON string)
func Process(requestBody []interface{}, dataStruct interface{}, initContext map[string]interface{}) (string, error) {
	// dataStruct argument must be Struct
	if reflect.TypeOf(dataStruct).Kind() != reflect.Struct {
		return "", fmt.Errorf("dataStruct argument must be instance of struct")
	}

	// Start recursion to process all fields in the request
	i, err := recursiveProcessRequest(requestBody, initContext, []string{}, dataStruct)
	if err != nil {
		return "", err
	}

	// Converting result to JSON string and return
	q, err := json.Marshal(i)
	if err != nil {
		return "", fmt.Errorf("JSON converting error")
	}

	return string(q), nil
}
