package analyse

var builtinClassNames = map[string]struct{}{
	"arrayaccess":              {},
	"arrayiterator":            {},
	"arrayobject":              {},
	"closure":                  {},
	"countable":                {},
	"dateinterval":             {},
	"datetime":                 {},
	"datetimeimmutable":        {},
	"datetimeinterface":        {},
	"datetimezone":             {},
	"badfunctioncallexception": {},
	"badmethodcallexception":   {},
	"domainexception":          {},
	"error":                    {},
	"errorexception":           {},
	"exception":                {},
	"generator":                {},
	"invalidargumentexception": {},
	"lengthexception":          {},
	"logicexception":           {},
	"iterator":                 {},
	"iteratoraggregate":        {},
	"jsonexception":            {},
	"jsonserializable":         {},
	"outofboundsexception":     {},
	"outofrangeexception":      {},
	"overflowexception":        {},
	"rangeexception":           {},
	"reflectionclass":          {},
	"reflectionexception":      {},
	"reflectionfunction":       {},
	"reflectionmethod":         {},
	"reflectionnamedtype":      {},
	"reflectionobject":         {},
	"reflectionparameter":      {},
	"reflectionproperty":       {},
	"sensitiveparameter":       {},
	"runtimeexception":         {},
	"simplexmlelement":         {},
	"splfixedarray":            {},
	"stdclass":                 {},
	"stringable":               {},
	"throwable":                {},
	"traversable":              {},
	"underflowexception":       {},
	"unexpectedvalueexception": {},
	"unitenum":                 {},
	"backedenum":               {},
	"valueerror":               {},
}

func isBuiltinClassName(name string) bool {
	_, ok := builtinClassNames[indexKey(name)]
	return ok
}

func (idx *ProjectIndex) seedBuiltins() {
	for _, class := range []ResolvedClass{
		{Name: "stdClass", Kind: "class"},
		{Name: "Exception", Kind: "class", Extends: []string{"Throwable"}},
		{Name: "ErrorException", Kind: "class", Extends: []string{"Exception"}},
		{Name: "LogicException", Kind: "class", Extends: []string{"Exception"}},
		{Name: "BadFunctionCallException", Kind: "class", Extends: []string{"LogicException"}},
		{Name: "BadMethodCallException", Kind: "class", Extends: []string{"BadFunctionCallException"}},
		{Name: "DomainException", Kind: "class", Extends: []string{"LogicException"}},
		{Name: "InvalidArgumentException", Kind: "class", Extends: []string{"LogicException"}},
		{Name: "LengthException", Kind: "class", Extends: []string{"LogicException"}},
		{Name: "OutOfRangeException", Kind: "class", Extends: []string{"LogicException"}},
		{Name: "RuntimeException", Kind: "class", Extends: []string{"Exception"}},
		{Name: "OutOfBoundsException", Kind: "class", Extends: []string{"RuntimeException"}},
		{Name: "OverflowException", Kind: "class", Extends: []string{"RuntimeException"}},
		{Name: "RangeException", Kind: "class", Extends: []string{"RuntimeException"}},
		{Name: "UnderflowException", Kind: "class", Extends: []string{"RuntimeException"}},
		{Name: "UnexpectedValueException", Kind: "class", Extends: []string{"RuntimeException"}},
		{Name: "Throwable", Kind: "interface"},
		{Name: "Error", Kind: "class", Extends: []string{"Throwable"}},
		{Name: "SplFixedArray", Kind: "class", Implements: []string{"ArrayAccess", "Countable", "Iterator"}},
		{Name: "DateTime", Kind: "class", Implements: []string{"DateTimeInterface"}},
		{Name: "DateTimeImmutable", Kind: "class", Implements: []string{"DateTimeInterface"}},
		{Name: "DateTimeInterface", Kind: "interface"},
		{Name: "DateTimeZone", Kind: "class"},
		{Name: "Closure", Kind: "class", Final: true},
		{Name: "Stringable", Kind: "interface"},
		{Name: "ReflectionClass", Kind: "class"},
		{Name: "ReflectionException", Kind: "class", Extends: []string{"Exception"}},
		{Name: "ReflectionFunctionAbstract", Kind: "class", Abstract: true},
		{Name: "ReflectionFunction", Kind: "class", Extends: []string{"ReflectionFunctionAbstract"}},
		{Name: "ReflectionMethod", Kind: "class", Extends: []string{"ReflectionFunctionAbstract"}},
		{Name: "ReflectionNamedType", Kind: "class", Extends: []string{"ReflectionType"}},
		{Name: "ReflectionObject", Kind: "class", Extends: []string{"ReflectionClass"}},
		{Name: "ReflectionParameter", Kind: "class"},
		{Name: "ReflectionProperty", Kind: "class"},
		{Name: "ReflectionType", Kind: "class"},
		{Name: "ArrayAccess", Kind: "interface"},
		{Name: "ArrayIterator", Kind: "class", Implements: []string{"Iterator", "Traversable"}},
		{Name: "ArrayObject", Kind: "class", Implements: []string{"IteratorAggregate", "Traversable"}},
		{Name: "Countable", Kind: "interface"},
		{Name: "DateInterval", Kind: "class"},
		{Name: "Generator", Kind: "class", Final: true},
		{Name: "Iterator", Kind: "interface", Extends: []string{"Traversable"}},
		{Name: "IteratorAggregate", Kind: "interface", Extends: []string{"Traversable"}},
		{Name: "JsonException", Kind: "class", Extends: []string{"Exception"}},
		{Name: "JsonSerializable", Kind: "interface"},
		{Name: "SensitiveParameter", Kind: "class", Final: true},
		{Name: "SimpleXMLElement", Kind: "class"},
		{Name: "Traversable", Kind: "interface"},
		{Name: "ValueError", Kind: "class", Extends: []string{"Error"}},
	} {
		if _, exists := idx.Classes[indexKey(class.Name)]; exists {
			continue
		}
		class.ID = stableSymbolID("class", "", class.Name)
		idx.Classes[indexKey(class.Name)] = class
	}
	for _, className := range []string{"DateTime", "DateTimeImmutable"} {
		idx.addMethodIfMissing(className, ResolvedMethod{Name: "createFromFormat", DeclaringClass: className, ReturnType: className + "|false", Params: []ResolvedParam{{Name: "format"}, {Name: "datetime"}, {Name: "timezone", HasDefault: true}}, Visibility: "public", IsStatic: true})
		idx.addMethodIfMissing(className, ResolvedMethod{Name: "createFromInterface", DeclaringClass: className, ReturnType: className, Params: []ResolvedParam{{Name: "object"}}, Visibility: "public", IsStatic: true})
		idx.addMethodIfMissing(className, ResolvedMethod{Name: "getLastErrors", DeclaringClass: className, ReturnType: "array|false", Visibility: "public", IsStatic: true})
	}
	idx.addMethodIfMissing("DateTime", ResolvedMethod{Name: "createFromImmutable", DeclaringClass: "DateTime", ReturnType: "DateTime", Params: []ResolvedParam{{Name: "object"}}, Visibility: "public", IsStatic: true})
	idx.addMethodIfMissing("DateTimeImmutable", ResolvedMethod{Name: "createFromMutable", DeclaringClass: "DateTimeImmutable", ReturnType: "DateTimeImmutable", Params: []ResolvedParam{{Name: "object"}}, Visibility: "public", IsStatic: true})
	dateTimeConstruct := []ResolvedParam{{Name: "datetime", HasDefault: true}, {Name: "timezone", HasDefault: true}}
	idx.addMethodIfMissing("DateTime", ResolvedMethod{Name: "__construct", DeclaringClass: "DateTime", Params: dateTimeConstruct, Visibility: "public"})
	idx.addMethodIfMissing("DateTimeImmutable", ResolvedMethod{Name: "__construct", DeclaringClass: "DateTimeImmutable", Params: dateTimeConstruct, Visibility: "public"})
	for _, method := range []ResolvedMethod{
		{Name: "format", ReturnType: "string", Params: []ResolvedParam{{Name: "format"}}, Visibility: "public"},
		{Name: "getTimestamp", ReturnType: "int", Visibility: "public"},
		{Name: "diff", ReturnType: "DateInterval", Params: []ResolvedParam{{Name: "targetObject"}, {Name: "absolute", HasDefault: true}}, Visibility: "public"},
		{Name: "getOffset", ReturnType: "int", Visibility: "public"},
		{Name: "getTimezone", ReturnType: "DateTimeZone|false", Visibility: "public"},
	} {
		idx.addMethodIfMissing("DateTimeInterface", method)
	}
	for _, className := range []string{"DateTime", "DateTimeImmutable"} {
		for _, method := range []ResolvedMethod{
			{Name: "modify", ReturnType: className, Params: []ResolvedParam{{Name: "modifier"}}, Visibility: "public"},
			{Name: "add", ReturnType: className, Params: []ResolvedParam{{Name: "interval"}}, Visibility: "public"},
			{Name: "sub", ReturnType: className, Params: []ResolvedParam{{Name: "interval"}}, Visibility: "public"},
			{Name: "setDate", ReturnType: className, Params: []ResolvedParam{{Name: "year"}, {Name: "month"}, {Name: "day"}}, Visibility: "public"},
			{Name: "setTime", ReturnType: className, Params: []ResolvedParam{{Name: "hour"}, {Name: "minute"}, {Name: "second", HasDefault: true}, {Name: "microsecond", HasDefault: true}}, Visibility: "public"},
			{Name: "setTimestamp", ReturnType: className, Params: []ResolvedParam{{Name: "timestamp"}}, Visibility: "public"},
			{Name: "setTimezone", ReturnType: className, Params: []ResolvedParam{{Name: "timezone"}}, Visibility: "public"},
		} {
			idx.addMethodIfMissing(className, method)
		}
	}
	for _, method := range []ResolvedMethod{
		{Name: "getMessage", ReturnType: "string", Visibility: "public"},
		{Name: "getCode", ReturnType: "int", Visibility: "public"},
		{Name: "getPrevious", ReturnType: "Throwable|null", Visibility: "public"},
		{Name: "getFile", ReturnType: "string", Visibility: "public"},
		{Name: "getLine", ReturnType: "int", Visibility: "public"},
		{Name: "getTrace", ReturnType: "array", Visibility: "public"},
		{Name: "getTraceAsString", ReturnType: "string", Visibility: "public"},
	} {
		idx.addMethodIfMissing("Throwable", method)
	}
	for _, method := range []ResolvedMethod{
		{Name: "getAttributes", ReturnType: "array", Params: []ResolvedParam{{Name: "name", HasDefault: true}, {Name: "flags", HasDefault: true}}, Visibility: "public"},
		{Name: "getConstant", ReturnType: "mixed", Params: []ResolvedParam{{Name: "name"}}, Visibility: "public"},
		{Name: "getConstructor", ReturnType: "ReflectionMethod|null", Visibility: "public"},
		{Name: "getMethod", ReturnType: "ReflectionMethod", Params: []ResolvedParam{{Name: "name"}}, Visibility: "public"},
		{Name: "getProperties", ReturnType: "array", Params: []ResolvedParam{{Name: "filter", HasDefault: true}}, Visibility: "public"},
		{Name: "getProperty", ReturnType: "ReflectionProperty", Params: []ResolvedParam{{Name: "name"}}, Visibility: "public"},
		{Name: "hasMethod", ReturnType: "bool", Params: []ResolvedParam{{Name: "name"}}, Visibility: "public"},
		{Name: "isFinal", ReturnType: "bool", Visibility: "public"},
		{Name: "isReadOnly", ReturnType: "bool", Visibility: "public"},
		{Name: "isSubclassOf", ReturnType: "bool", Params: []ResolvedParam{{Name: "class"}}, Visibility: "public"},
	} {
		idx.addMethodIfMissing("ReflectionClass", method)
	}
	for _, method := range []ResolvedMethod{
		{Name: "getAttributes", ReturnType: "array", Params: []ResolvedParam{{Name: "name", HasDefault: true}, {Name: "flags", HasDefault: true}}, Visibility: "public"},
		{Name: "getValue", ReturnType: "mixed", Params: []ResolvedParam{{Name: "object", HasDefault: true}}, Visibility: "public"},
		{Name: "setValue", Params: []ResolvedParam{{Name: "objectOrValue"}, {Name: "value", HasDefault: true}}, Visibility: "public"},
		{Name: "isReadOnly", ReturnType: "bool", Visibility: "public"},
	} {
		idx.addMethodIfMissing("ReflectionProperty", method)
	}
	for _, method := range []ResolvedMethod{
		{Name: "getName", ReturnType: "string", Visibility: "public"},
		{Name: "getNumberOfParameters", ReturnType: "int", Visibility: "public"},
		{Name: "getParameters", ReturnType: "array", Visibility: "public"},
		{Name: "getReturnType", ReturnType: "ReflectionType|null", Visibility: "public"},
		{Name: "hasReturnType", ReturnType: "bool", Visibility: "public"},
		{Name: "getDocComment", ReturnType: "string|false", Visibility: "public"},
		{Name: "getFileName", ReturnType: "string|false", Visibility: "public"},
		{Name: "isDeprecated", ReturnType: "bool", Visibility: "public"},
		{Name: "isVariadic", ReturnType: "bool", Visibility: "public"},
		{Name: "isGenerator", ReturnType: "bool", Visibility: "public"},
		{Name: "returnsReference", ReturnType: "bool", Visibility: "public"},
	} {
		idx.addMethodIfMissing("ReflectionFunctionAbstract", method)
	}
	for _, method := range []ResolvedMethod{
		{Name: "isPublic", ReturnType: "bool", Visibility: "public"},
		{Name: "isPrivate", ReturnType: "bool", Visibility: "public"},
		{Name: "isProtected", ReturnType: "bool", Visibility: "public"},
		{Name: "isStatic", ReturnType: "bool", Visibility: "public"},
		{Name: "isAbstract", ReturnType: "bool", Visibility: "public"},
		{Name: "isFinal", ReturnType: "bool", Visibility: "public"},
		{Name: "isConstructor", ReturnType: "bool", Visibility: "public"},
		{Name: "isDestructor", ReturnType: "bool", Visibility: "public"},
		{Name: "getDeclaringClass", ReturnType: "ReflectionClass", Visibility: "public"},
		{Name: "invokeArgs", ReturnType: "mixed", Params: []ResolvedParam{{Name: "object"}, {Name: "args"}}, Visibility: "public"},
		{Name: "setAccessible", Params: []ResolvedParam{{Name: "accessible"}}, Visibility: "public"},
	} {
		idx.addMethodIfMissing("ReflectionMethod", method)
	}
	idx.addMethodIfMissing("ReflectionMethod", ResolvedMethod{Name: "invoke", ReturnType: "mixed", Params: []ResolvedParam{{Name: "object"}, {Name: "args", IsVariadic: true}}, Visibility: "public"})
	for _, method := range []ResolvedMethod{
		{Name: "getName", ReturnType: "string", Visibility: "public"},
		{Name: "getType", ReturnType: "ReflectionType|null", Visibility: "public"},
		{Name: "hasType", ReturnType: "bool", Visibility: "public"},
		{Name: "isOptional", ReturnType: "bool", Visibility: "public"},
		{Name: "isPassedByReference", ReturnType: "bool", Visibility: "public"},
		{Name: "isDefaultValueAvailable", ReturnType: "bool", Visibility: "public"},
		{Name: "getDefaultValue", ReturnType: "mixed", Visibility: "public"},
		{Name: "allowsNull", ReturnType: "bool", Visibility: "public"},
		{Name: "isVariadic", ReturnType: "bool", Visibility: "public"},
		{Name: "isPromoted", ReturnType: "bool", Visibility: "public"},
	} {
		idx.addMethodIfMissing("ReflectionParameter", method)
	}
	idx.addMethodIfMissing("ReflectionType", ResolvedMethod{Name: "allowsNull", ReturnType: "bool", Visibility: "public"})
	idx.addMethodIfMissing("ReflectionNamedType", ResolvedMethod{Name: "getName", ReturnType: "string", Visibility: "public"})
	idx.addMethodIfMissing("ReflectionNamedType", ResolvedMethod{Name: "isBuiltin", ReturnType: "bool", Visibility: "public"})
	idx.addMethodIfMissing("Closure", ResolvedMethod{Name: "fromCallable", DeclaringClass: "Closure", ReturnType: "Closure", Params: []ResolvedParam{{Name: "callback"}}, Visibility: "public", IsStatic: true})
	idx.addMethodIfMissing("DateTimeZone", ResolvedMethod{Name: "__construct", DeclaringClass: "DateTimeZone", Params: []ResolvedParam{{Name: "timezone"}}, Visibility: "public"})
	idx.addMethodIfMissing("DateInterval", ResolvedMethod{Name: "__construct", DeclaringClass: "DateInterval", Params: []ResolvedParam{{Name: "duration"}}, Visibility: "public"})
	idx.addMethodIfMissing("ArrayObject", ResolvedMethod{Name: "__construct", DeclaringClass: "ArrayObject", Params: []ResolvedParam{{Name: "array", HasDefault: true}, {Name: "flags", HasDefault: true}, {Name: "iteratorClass", HasDefault: true}}, Visibility: "public"})
	idx.addMethodIfMissing("ArrayIterator", ResolvedMethod{Name: "__construct", DeclaringClass: "ArrayIterator", Params: []ResolvedParam{{Name: "array", HasDefault: true}, {Name: "flags", HasDefault: true}}, Visibility: "public"})
	idx.addMethodIfMissing("SplFixedArray", ResolvedMethod{Name: "__construct", DeclaringClass: "SplFixedArray", Params: []ResolvedParam{{Name: "size", HasDefault: true}}, Visibility: "public"})
	idx.addMethodIfMissing("ErrorException", ResolvedMethod{Name: "__construct", DeclaringClass: "ErrorException", Params: []ResolvedParam{{Name: "message", HasDefault: true}, {Name: "code", HasDefault: true}, {Name: "severity", HasDefault: true}, {Name: "filename", HasDefault: true}, {Name: "line", HasDefault: true}, {Name: "previous", HasDefault: true}}, Visibility: "public"})
	idx.addMethodIfMissing("Error", ResolvedMethod{Name: "__construct", DeclaringClass: "Error", Params: []ResolvedParam{{Name: "message", HasDefault: true}, {Name: "code", HasDefault: true}, {Name: "previous", HasDefault: true}}, Visibility: "public"})
	idx.addMethodIfMissing("Exception", ResolvedMethod{Name: "__construct", DeclaringClass: "Exception", Params: []ResolvedParam{{Name: "message", HasDefault: true}, {Name: "code", HasDefault: true}, {Name: "previous", HasDefault: true}}, Visibility: "public"})
	idx.addMethodIfMissing("ReflectionClass", ResolvedMethod{Name: "__construct", DeclaringClass: "ReflectionClass", Params: []ResolvedParam{{Name: "objectOrClass"}}, Visibility: "public"})
	idx.addMethodIfMissing("ReflectionMethod", ResolvedMethod{Name: "__construct", DeclaringClass: "ReflectionMethod", Params: []ResolvedParam{{Name: "objectOrMethod"}, {Name: "method", HasDefault: true}}, Visibility: "public"})
	idx.addMethodIfMissing("ReflectionProperty", ResolvedMethod{Name: "__construct", DeclaringClass: "ReflectionProperty", Params: []ResolvedParam{{Name: "class"}, {Name: "property"}}, Visibility: "public"})
	idx.addMethodIfMissing("ReflectionObject", ResolvedMethod{Name: "__construct", DeclaringClass: "ReflectionObject", Params: []ResolvedParam{{Name: "object"}}, Visibility: "public"})
	for _, constant := range []string{"ATOM", "COOKIE", "ISO8601", "RFC822", "RFC850", "RFC1036", "RFC1123", "RFC7231", "RFC2822", "RFC3339", "RFC3339_EXTENDED", "RSS", "W3C"} {
		idx.addClassConstantIfMissing("DateTime", ResolvedConstant{Name: constant, DeclaringClass: "DateTime", Visibility: "public"})
		idx.addClassConstantIfMissing("DateTimeImmutable", ResolvedConstant{Name: constant, DeclaringClass: "DateTimeImmutable", Visibility: "public"})
		idx.addClassConstantIfMissing("DateTimeInterface", ResolvedConstant{Name: constant, DeclaringClass: "DateTimeInterface", Visibility: "public"})
	}
	for _, fn := range []ResolvedFunction{
		{Name: "abs", Params: []ResolvedParam{{Name: "num"}}},
		{Name: "addslashes", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "array_any", Params: []ResolvedParam{{Name: "array"}, {Name: "callback"}}},
		{Name: "array_chunk", Params: []ResolvedParam{{Name: "array"}, {Name: "length"}, {Name: "preserve_keys", HasDefault: true}}},
		{Name: "array_combine", Params: []ResolvedParam{{Name: "keys"}, {Name: "values"}}},
		{Name: "array_column", Params: []ResolvedParam{{Name: "array"}, {Name: "column_key"}, {Name: "index_key", HasDefault: true}}},
		{Name: "array_diff", Params: []ResolvedParam{{Name: "array"}, {Name: "arrays", IsVariadic: true}}},
		{Name: "array_fill", Params: []ResolvedParam{{Name: "start_index"}, {Name: "count"}, {Name: "value"}}},
		{Name: "array_fill_keys", Params: []ResolvedParam{{Name: "keys"}, {Name: "value"}}},
		{Name: "array_filter", Params: []ResolvedParam{{Name: "array"}, {Name: "callback", HasDefault: true}, {Name: "mode", HasDefault: true}}},
		{Name: "array_flip", Params: []ResolvedParam{{Name: "array"}}},
		{Name: "array_intersect_assoc", Params: []ResolvedParam{{Name: "array"}, {Name: "arrays", IsVariadic: true}}},
		{Name: "array_intersect_key", Params: []ResolvedParam{{Name: "array"}, {Name: "arrays", IsVariadic: true}}},
		{Name: "array_key_exists", Params: []ResolvedParam{{Name: "key"}, {Name: "array"}}},
		{Name: "array_key_first", Params: []ResolvedParam{{Name: "array"}}},
		{Name: "array_map", Params: []ResolvedParam{{Name: "callback"}, {Name: "array"}, {Name: "arrays", IsVariadic: true}}},
		{Name: "array_merge", Params: []ResolvedParam{{Name: "arrays", IsVariadic: true}}},
		{Name: "array_merge_recursive", Params: []ResolvedParam{{Name: "arrays", IsVariadic: true}}},
		{Name: "array_pop", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "array_push", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "values", IsVariadic: true}}},
		{Name: "array_reduce", Params: []ResolvedParam{{Name: "array"}, {Name: "callback"}, {Name: "initial", HasDefault: true}}},
		{Name: "array_replace_recursive", Params: []ResolvedParam{{Name: "array"}, {Name: "replacements", IsVariadic: true}}},
		{Name: "array_search", Params: []ResolvedParam{{Name: "needle"}, {Name: "haystack"}, {Name: "strict", HasDefault: true}}},
		{Name: "array_shift", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "array_splice", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "offset"}, {Name: "length", HasDefault: true}, {Name: "replacement", HasDefault: true}}},
		{Name: "array_keys", Params: []ResolvedParam{{Name: "array"}, {Name: "filter_value", HasDefault: true}, {Name: "strict", HasDefault: true}}},
		{Name: "array_slice", Params: []ResolvedParam{{Name: "array"}, {Name: "offset"}, {Name: "length", HasDefault: true}, {Name: "preserve_keys", HasDefault: true}}},
		{Name: "array_unique", Params: []ResolvedParam{{Name: "array"}, {Name: "flags", HasDefault: true}}},
		{Name: "array_unshift", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "values", IsVariadic: true}}},
		{Name: "array_sum", Params: []ResolvedParam{{Name: "array"}}},
		{Name: "array_values", Params: []ResolvedParam{{Name: "array"}}},
		{Name: "array_walk", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "callback"}, {Name: "arg", HasDefault: true}}},
		{Name: "array_walk_recursive", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "callback"}, {Name: "arg", HasDefault: true}}},
		{Name: "arsort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "flags", HasDefault: true}}},
		{Name: "asort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "flags", HasDefault: true}}},
		{Name: "assert", Params: []ResolvedParam{{Name: "assertion"}, {Name: "description", HasDefault: true}}},
		{Name: "base64_decode", Params: []ResolvedParam{{Name: "string"}, {Name: "strict", HasDefault: true}}},
		{Name: "base64_encode", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "basename", Params: []ResolvedParam{{Name: "path"}, {Name: "suffix", HasDefault: true}}},
		{Name: "bin2hex", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "ceil", Params: []ResolvedParam{{Name: "num"}}},
		{Name: "checkdate", Params: []ResolvedParam{{Name: "month"}, {Name: "day"}, {Name: "year"}}},
		{Name: "class_exists", Params: []ResolvedParam{{Name: "class"}, {Name: "autoload", HasDefault: true}}},
		{Name: "compact", Params: []ResolvedParam{{Name: "var_name"}, {Name: "var_names", IsVariadic: true}}},
		{Name: "constant", Params: []ResolvedParam{{Name: "name"}}},
		{Name: "count", Params: []ResolvedParam{{Name: "value"}, {Name: "mode", HasDefault: true}}},
		{Name: "crc32", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "date", Params: []ResolvedParam{{Name: "format"}, {Name: "timestamp", HasDefault: true}}},
		{Name: "define", Params: []ResolvedParam{{Name: "constant_name"}, {Name: "value"}, {Name: "case_insensitive", HasDefault: true}}},
		{Name: "defined", Params: []ResolvedParam{{Name: "constant_name"}}},
		{Name: "die", Params: []ResolvedParam{{Name: "status", HasDefault: true}}},
		{Name: "dirname", Params: []ResolvedParam{{Name: "path"}, {Name: "levels", HasDefault: true}}},
		{Name: "empty", Params: []ResolvedParam{{Name: "var"}}},
		{Name: "enum_exists", Params: []ResolvedParam{{Name: "enum"}, {Name: "autoload", HasDefault: true}}},
		{Name: "error_log", Params: []ResolvedParam{{Name: "message"}, {Name: "message_type", HasDefault: true}, {Name: "destination", HasDefault: true}, {Name: "additional_headers", HasDefault: true}}},
		{Name: "end", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "eval", Params: []ResolvedParam{{Name: "code"}}},
		{Name: "exec", Params: []ResolvedParam{{Name: "command"}, {Name: "output", HasDefault: true, IsByRef: true, IsOut: true}, {Name: "result_code", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "exit", Params: []ResolvedParam{{Name: "status", HasDefault: true}}},
		{Name: "explode", Params: []ResolvedParam{{Name: "separator"}, {Name: "string"}, {Name: "limit", HasDefault: true}}},
		{Name: "extract", Params: []ResolvedParam{{Name: "array"}, {Name: "flags", HasDefault: true}, {Name: "prefix", HasDefault: true}}},
		{Name: "extension_loaded", Params: []ResolvedParam{{Name: "extension"}}},
		{Name: "fclose", Params: []ResolvedParam{{Name: "stream"}}},
		{Name: "fgetcsv", Params: []ResolvedParam{{Name: "stream"}, {Name: "length", HasDefault: true}, {Name: "separator", HasDefault: true}, {Name: "enclosure", HasDefault: true}, {Name: "escape", HasDefault: true}}},
		{Name: "file_exists", Params: []ResolvedParam{{Name: "filename"}}},
		{Name: "file_get_contents", Params: []ResolvedParam{{Name: "filename"}, {Name: "use_include_path", HasDefault: true}, {Name: "context", HasDefault: true}, {Name: "offset", HasDefault: true}, {Name: "length", HasDefault: true}}},
		{Name: "file_put_contents", Params: []ResolvedParam{{Name: "filename"}, {Name: "data"}, {Name: "flags", HasDefault: true}, {Name: "context", HasDefault: true}}},
		{Name: "filesize", Params: []ResolvedParam{{Name: "filename"}}},
		{Name: "filter_var", Params: []ResolvedParam{{Name: "value"}, {Name: "filter", HasDefault: true}, {Name: "options", HasDefault: true}}},
		{Name: "floor", Params: []ResolvedParam{{Name: "num"}}},
		{Name: "fopen", ReturnType: "resource|false", Params: []ResolvedParam{{Name: "filename"}, {Name: "mode"}, {Name: "use_include_path", HasDefault: true}, {Name: "context", HasDefault: true}}},
		{Name: "fpassthru", Params: []ResolvedParam{{Name: "stream"}}},
		{Name: "fputcsv", Params: []ResolvedParam{{Name: "stream"}, {Name: "fields"}, {Name: "separator", HasDefault: true}, {Name: "enclosure", HasDefault: true}, {Name: "escape", HasDefault: true}, {Name: "eol", HasDefault: true}}},
		{Name: "fscanf", Params: []ResolvedParam{{Name: "stream"}, {Name: "format"}, {Name: "vars", HasDefault: true, IsVariadic: true, IsByRef: true, IsOut: true}}},
		{Name: "func_get_args"},
		{Name: "func_num_args"},
		{Name: "function_exists", Params: []ResolvedParam{{Name: "function"}}},
		{Name: "get_class", Params: []ResolvedParam{{Name: "object", HasDefault: true}}},
		{Name: "get_object_vars", Params: []ResolvedParam{{Name: "object"}}},
		{Name: "getenv", Params: []ResolvedParam{{Name: "name", HasDefault: true}, {Name: "local_only", HasDefault: true}}},
		{Name: "getopt", Params: []ResolvedParam{{Name: "short_options"}, {Name: "long_options", HasDefault: true}, {Name: "rest_index", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "glob", Params: []ResolvedParam{{Name: "pattern"}, {Name: "flags", HasDefault: true}}},
		{Name: "hash", Params: []ResolvedParam{{Name: "algo"}, {Name: "data"}, {Name: "binary", HasDefault: true}, {Name: "options", HasDefault: true}}},
		{Name: "headers_sent", Params: []ResolvedParam{{Name: "filename", HasDefault: true, IsByRef: true, IsOut: true}, {Name: "line", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "hrtime", Params: []ResolvedParam{{Name: "as_number", HasDefault: true}}},
		{Name: "http_build_query", Params: []ResolvedParam{{Name: "data"}, {Name: "numeric_prefix", HasDefault: true}, {Name: "arg_separator", HasDefault: true}, {Name: "encoding_type", HasDefault: true}}},
		{Name: "htmlspecialchars", Params: []ResolvedParam{{Name: "string"}, {Name: "flags", HasDefault: true}, {Name: "encoding", HasDefault: true}, {Name: "double_encode", HasDefault: true}}},
		{Name: "implode", Params: []ResolvedParam{{Name: "separator"}, {Name: "array", HasDefault: true}}},
		{Name: "in_array", Params: []ResolvedParam{{Name: "needle"}, {Name: "haystack"}, {Name: "strict", HasDefault: true}}},
		{Name: "intdiv", Params: []ResolvedParam{{Name: "num1"}, {Name: "num2"}}},
		{Name: "intval", Params: []ResolvedParam{{Name: "value"}, {Name: "base", HasDefault: true}}},
		{Name: "interface_exists", Params: []ResolvedParam{{Name: "interface"}, {Name: "autoload", HasDefault: true}}},
		{Name: "isset", Params: []ResolvedParam{{Name: "var"}, {Name: "vars", IsVariadic: true}}},
		{Name: "is_array", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_bool", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_callable", Params: []ResolvedParam{{Name: "value"}, {Name: "syntax_only", HasDefault: true}, {Name: "callable_name", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "is_countable", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_dir", Params: []ResolvedParam{{Name: "filename"}}},
		{Name: "is_file", Params: []ResolvedParam{{Name: "filename"}}},
		{Name: "is_finite", Params: []ResolvedParam{{Name: "num"}}},
		{Name: "is_float", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_int", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_null", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_numeric", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_object", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_resource", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_scalar", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_string", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "iterator_count", Params: []ResolvedParam{{Name: "iterator"}}},
		{Name: "iterator_to_array", Params: []ResolvedParam{{Name: "iterator"}, {Name: "preserve_keys", HasDefault: true}}},
		{Name: "json_last_error"},
		{Name: "json_decode", Params: []ResolvedParam{{Name: "json"}, {Name: "associative", HasDefault: true}, {Name: "depth", HasDefault: true}, {Name: "flags", HasDefault: true}}},
		{Name: "json_encode", Params: []ResolvedParam{{Name: "value"}, {Name: "flags", HasDefault: true}, {Name: "depth", HasDefault: true}}},
		{Name: "krsort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "flags", HasDefault: true}}},
		{Name: "ksort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "flags", HasDefault: true}}},
		{Name: "lcfirst", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "libxml_clear_errors"},
		{Name: "libxml_get_errors"},
		{Name: "libxml_use_internal_errors", Params: []ResolvedParam{{Name: "use_errors", HasDefault: true}}},
		{Name: "ltrim", Params: []ResolvedParam{{Name: "string"}, {Name: "characters", HasDefault: true}}},
		{Name: "max", Params: []ResolvedParam{{Name: "value"}, {Name: "values", IsVariadic: true}}},
		{Name: "mb_strtoupper", Params: []ResolvedParam{{Name: "string"}, {Name: "encoding", HasDefault: true}}},
		{Name: "mb_strlen", Params: []ResolvedParam{{Name: "string"}, {Name: "encoding", HasDefault: true}}},
		{Name: "mb_substr", Params: []ResolvedParam{{Name: "string"}, {Name: "start"}, {Name: "length", HasDefault: true}, {Name: "encoding", HasDefault: true}}},
		{Name: "md5", Params: []ResolvedParam{{Name: "string"}, {Name: "binary", HasDefault: true}}},
		{Name: "method_exists", Params: []ResolvedParam{{Name: "object_or_class"}, {Name: "method"}}},
		{Name: "microtime", Params: []ResolvedParam{{Name: "as_float", HasDefault: true}}},
		{Name: "min", Params: []ResolvedParam{{Name: "value"}, {Name: "values", IsVariadic: true}}},
		{Name: "mkdir", Params: []ResolvedParam{{Name: "directory"}, {Name: "permissions", HasDefault: true}, {Name: "recursive", HasDefault: true}, {Name: "context", HasDefault: true}}},
		{Name: "mktime", Params: []ResolvedParam{{Name: "hour", HasDefault: true}, {Name: "minute", HasDefault: true}, {Name: "second", HasDefault: true}, {Name: "month", HasDefault: true}, {Name: "day", HasDefault: true}, {Name: "year", HasDefault: true}}},
		{Name: "natcasesort", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "natsort", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "next", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "number_format", Params: []ResolvedParam{{Name: "num"}, {Name: "decimals", HasDefault: true}, {Name: "decimal_separator", HasDefault: true}, {Name: "thousands_separator", HasDefault: true}}},
		{Name: "ob_start", Params: []ResolvedParam{{Name: "callback", HasDefault: true}, {Name: "chunk_size", HasDefault: true}, {Name: "flags", HasDefault: true}}},
		{Name: "parse_str", Params: []ResolvedParam{{Name: "string"}, {Name: "result", IsByRef: true, IsOut: true}}},
		{Name: "parse_url", Params: []ResolvedParam{{Name: "url"}, {Name: "component", HasDefault: true}}},
		{Name: "pathinfo", Params: []ResolvedParam{{Name: "path"}, {Name: "flags", HasDefault: true}}},
		{Name: "passthru", Params: []ResolvedParam{{Name: "command"}, {Name: "result_code", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "preg_filter", Params: []ResolvedParam{{Name: "pattern"}, {Name: "replacement"}, {Name: "subject"}, {Name: "limit", HasDefault: true}, {Name: "count", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "preg_match", Params: []ResolvedParam{{Name: "pattern"}, {Name: "subject"}, {Name: "matches", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "preg_match_all", Params: []ResolvedParam{{Name: "pattern"}, {Name: "subject"}, {Name: "matches", HasDefault: true, IsByRef: true, IsOut: true}, {Name: "flags", HasDefault: true}, {Name: "offset", HasDefault: true}}},
		{Name: "preg_quote", Params: []ResolvedParam{{Name: "str"}, {Name: "delimiter", HasDefault: true}}},
		{Name: "preg_replace", Params: []ResolvedParam{{Name: "pattern"}, {Name: "replacement"}, {Name: "subject"}, {Name: "limit", HasDefault: true}, {Name: "count", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "preg_replace_callback", Params: []ResolvedParam{{Name: "pattern"}, {Name: "callback"}, {Name: "subject"}, {Name: "limit", HasDefault: true}, {Name: "count", HasDefault: true, IsByRef: true, IsOut: true}, {Name: "flags", HasDefault: true}}},
		{Name: "preg_replace_callback_array", Params: []ResolvedParam{{Name: "pattern"}, {Name: "subject"}, {Name: "limit", HasDefault: true}, {Name: "count", HasDefault: true, IsByRef: true, IsOut: true}, {Name: "flags", HasDefault: true}}},
		{Name: "preg_split", Params: []ResolvedParam{{Name: "pattern"}, {Name: "subject"}, {Name: "limit", HasDefault: true}, {Name: "flags", HasDefault: true}}},
		{Name: "prev", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "printf", Params: []ResolvedParam{{Name: "format"}, {Name: "values", IsVariadic: true}}},
		{Name: "proc_open", Params: []ResolvedParam{{Name: "command"}, {Name: "descriptor_spec"}, {Name: "pipes", IsByRef: true, IsOut: true}, {Name: "cwd", HasDefault: true}, {Name: "env_vars", HasDefault: true}, {Name: "options", HasDefault: true}}},
		{Name: "putenv", Params: []ResolvedParam{{Name: "assignment"}}},
		{Name: "random_bytes", Params: []ResolvedParam{{Name: "length"}}},
		{Name: "random_int", Params: []ResolvedParam{{Name: "min"}, {Name: "max"}}},
		{Name: "reset", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "rewind", Params: []ResolvedParam{{Name: "stream"}}},
		{Name: "rmdir", Params: []ResolvedParam{{Name: "directory"}, {Name: "context", HasDefault: true}}},
		{Name: "range", Params: []ResolvedParam{{Name: "start"}, {Name: "end"}, {Name: "step", HasDefault: true}}},
		{Name: "round", Params: []ResolvedParam{{Name: "num"}, {Name: "precision", HasDefault: true}, {Name: "mode", HasDefault: true}}},
		{Name: "rsort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "flags", HasDefault: true}}},
		{Name: "rtrim", Params: []ResolvedParam{{Name: "string"}, {Name: "characters", HasDefault: true}}},
		{Name: "scandir", Params: []ResolvedParam{{Name: "directory"}, {Name: "sorting_order", HasDefault: true}, {Name: "context", HasDefault: true}}},
		{Name: "serialize", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "settype", Params: []ResolvedParam{{Name: "var", IsByRef: true}, {Name: "type"}}},
		{Name: "sha1", Params: []ResolvedParam{{Name: "string"}, {Name: "binary", HasDefault: true}}},
		{Name: "simplexml_load_file", Params: []ResolvedParam{{Name: "filename"}, {Name: "class_name", HasDefault: true}, {Name: "options", HasDefault: true}, {Name: "namespace_or_prefix", HasDefault: true}, {Name: "is_prefix", HasDefault: true}}},
		{Name: "shuffle", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "similar_text", Params: []ResolvedParam{{Name: "string1"}, {Name: "string2"}, {Name: "percent", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "sleep", Params: []ResolvedParam{{Name: "seconds"}}},
		{Name: "sort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "flags", HasDefault: true}}},
		{Name: "sprintf", Params: []ResolvedParam{{Name: "format"}, {Name: "values", IsVariadic: true}}},
		{Name: "sscanf", Params: []ResolvedParam{{Name: "string"}, {Name: "format"}, {Name: "vars", HasDefault: true, IsVariadic: true, IsByRef: true, IsOut: true}}},
		{Name: "stripos", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}, {Name: "offset", HasDefault: true}}},
		{Name: "str_contains", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}}},
		{Name: "str_ends_with", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}}},
		{Name: "str_pad", Params: []ResolvedParam{{Name: "string"}, {Name: "length"}, {Name: "pad_string", HasDefault: true}, {Name: "pad_type", HasDefault: true}}},
		{Name: "str_repeat", Params: []ResolvedParam{{Name: "string"}, {Name: "times"}}},
		{Name: "str_replace", Params: []ResolvedParam{{Name: "search"}, {Name: "replace"}, {Name: "subject"}, {Name: "count", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "str_starts_with", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}}},
		{Name: "strcasecmp", Params: []ResolvedParam{{Name: "string1"}, {Name: "string2"}}},
		{Name: "strcmp", Params: []ResolvedParam{{Name: "string1"}, {Name: "string2"}}},
		{Name: "strlen", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "strpos", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}, {Name: "offset", HasDefault: true}}},
		{Name: "strrpos", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}, {Name: "offset", HasDefault: true}}},
		{Name: "strtolower", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "strtoupper", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "stream_copy_to_stream", Params: []ResolvedParam{{Name: "from"}, {Name: "to"}, {Name: "length", HasDefault: true}, {Name: "offset", HasDefault: true}}},
		{Name: "stream_get_contents", Params: []ResolvedParam{{Name: "stream"}, {Name: "length", HasDefault: true}, {Name: "offset", HasDefault: true}}},
		{Name: "strtr", Params: []ResolvedParam{{Name: "string"}, {Name: "from"}, {Name: "to", HasDefault: true}}},
		{Name: "substr", Params: []ResolvedParam{{Name: "string"}, {Name: "offset"}, {Name: "length", HasDefault: true}}},
		{Name: "substr_count", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}, {Name: "offset", HasDefault: true}, {Name: "length", HasDefault: true}}},
		{Name: "sys_get_temp_dir"},
		{Name: "system", Params: []ResolvedParam{{Name: "command"}, {Name: "result_code", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "tempnam", Params: []ResolvedParam{{Name: "directory"}, {Name: "prefix"}}},
		{Name: "time"},
		{Name: "trait_exists", Params: []ResolvedParam{{Name: "trait"}, {Name: "autoload", HasDefault: true}}},
		{Name: "trim", Params: []ResolvedParam{{Name: "string"}, {Name: "characters", HasDefault: true}}},
		{Name: "trigger_error", Params: []ResolvedParam{{Name: "message"}, {Name: "error_level", HasDefault: true}}},
		{Name: "uasort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "callback"}}},
		{Name: "ucfirst", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "ucwords", Params: []ResolvedParam{{Name: "string"}, {Name: "separators", HasDefault: true}}},
		{Name: "uniqid", Params: []ResolvedParam{{Name: "prefix", HasDefault: true}, {Name: "more_entropy", HasDefault: true}}},
		{Name: "uksort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "callback"}}},
		{Name: "unlink", Params: []ResolvedParam{{Name: "filename"}, {Name: "context", HasDefault: true}}},
		{Name: "unserialize", Params: []ResolvedParam{{Name: "data"}, {Name: "options", HasDefault: true}}},
		{Name: "urlencode", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "usort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "callback"}}},
		{Name: "unset", Params: []ResolvedParam{{Name: "var"}, {Name: "vars", IsVariadic: true}}},
		{Name: "usleep", Params: []ResolvedParam{{Name: "microseconds"}}},
	} {
		idx.addFunction(fn)
	}
	for _, constant := range []string{"PHP_VERSION", "PHP_VERSION_ID", "PHP_MAJOR_VERSION", "PHP_MINOR_VERSION", "PHP_OS", "PHP_EOL", "true", "false", "null"} {
		idx.addGlobalConstant("", constant)
	}
}
