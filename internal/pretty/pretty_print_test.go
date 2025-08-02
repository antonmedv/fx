package pretty_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/pretty"
)

func stripEscapeSequences(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(s, "")
}

func TestPrettyPrint(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected string
		inline   bool
	}{
		{
			name:     "standalone null with inline",
			json:     `null`,
			expected: `null`,
			inline:   true,
		},
		{
			name:     "standalone null without inline",
			json:     `null`,
			expected: `null`,
			inline:   false,
		},
		{
			name:     "standalone true with inline",
			json:     `true`,
			expected: `true`,
			inline:   true,
		},
		{
			name:     "standalone true without inline",
			json:     `true`,
			expected: `true`,
			inline:   false,
		},
		{
			name:     "standalone false with inline",
			json:     `false`,
			expected: `false`,
			inline:   true,
		},
		{
			name:     "standalone false without inline",
			json:     `false`,
			expected: `false`,
			inline:   false,
		},
		{
			name: "array with empty object and empty array with inline",
			json: `[{}, []]`,
			expected: `[
  {},
  []
]`,
			inline: true,
		},
		{
			name: "array with empty object and empty array without inline",
			json: `[{}, []]`,
			expected: `[
  {},
  []
]`,
			inline: false,
		},
		{
			name: "simple object with inline",
			json: `{"name":"John","age":30,"city":"New York"}`,
			expected: `{
  "name": "John",
  "age": 30,
  "city": "New York"
}`,
			inline: true,
		},
		{
			name: "simple object without inline",
			json: `{"name":"John","age":30,"city":"New York"}`,
			expected: `{
  "name": "John",
  "age": 30,
  "city": "New York"
}`,
			inline: false,
		},
		{
			name: "nested object with inline",
			json: `{"person":{"name":"John","age":30,"address":{"city":"New York","zip":"10001"}}}`,
			expected: `{
  "person": {
    "name": "John",
    "age": 30,
    "address": {
      "city": "New York",
      "zip": "10001"
    }
  }
}`,
			inline: true,
		},
		{
			name: "nested object without inline",
			json: `{"person":{"name":"John","age":30,"address":{"city":"New York","zip":"10001"}}}`,
			expected: `{
  "person": {
    "name": "John",
    "age": 30,
    "address": {
      "city": "New York",
      "zip": "10001"
    }
  }
}`,
			inline: false,
		},
		{
			name: "array of numbers with inline",
			json: `{"numbers":[1,2,3,4,5]}`,
			expected: `{
  "numbers": [ 1, 2, 3, 4, 5 ]
}`,
			inline: true,
		},
		{
			name: "array of numbers without inline",
			json: `{"numbers":[1,2,3,4,5]}`,
			expected: `{
  "numbers": [
    1,
    2,
    3,
    4,
    5
  ]
}`,
			inline: false,
		},
		{
			name: "array of objects with inline",
			json: `{"people":[{"name":"John","age":30},{"name":"Jane","age":25}]}`,
			expected: `{
  "people": [
    { "name": "John", "age": 30 },
    { "name": "Jane", "age": 25 }
  ]
}`,
			inline: true,
		},
		{
			name: "array of objects without inline",
			json: `{"people":[{"name":"John","age":30},{"name":"Jane","age":25}]}`,
			expected: `{
  "people": [
    {
      "name": "John",
      "age": 30
    },
    {
      "name": "Jane",
      "age": 25
    }
  ]
}`,
			inline: false,
		},
		{
			name:     "empty object with inline",
			json:     `{}`,
			expected: `{}`,
			inline:   true,
		},
		{
			name:     "empty object without inline",
			json:     `{}`,
			expected: `{}`,
			inline:   false,
		},
		{
			name:     "empty array with inline",
			json:     `[]`,
			expected: `[]`,
			inline:   true,
		},
		{
			name:     "empty array without inline",
			json:     `[]`,
			expected: `[]`,
			inline:   false,
		},
		{
			name: "null value with inline",
			json: `{"value":null}`,
			expected: `{
  "value": null
}`,
			inline: true,
		},
		{
			name: "null value without inline",
			json: `{"value":null}`,
			expected: `{
  "value": null
}`,
			inline: false,
		},
		{
			name: "boolean values with inline",
			json: `{"active":true,"verified":false}`,
			expected: `{
  "active": true,
  "verified": false
}`,
			inline: true,
		},
		{
			name: "boolean values without inline",
			json: `{"active":true,"verified":false}`,
			expected: `{
  "active": true,
  "verified": false
}`,
			inline: false,
		},
		{
			name: "string with special characters with inline",
			json: `{"message":"Hello, \"World\"!\nNew line\tTab"}`,
			expected: `{
  "message": "Hello, \"World\"!\nNew line\tTab"
}`,
			inline: true,
		},
		{
			name: "string with special characters without inline",
			json: `{"message":"Hello, \"World\"!\nNew line\tTab"}`,
			expected: `{
  "message": "Hello, \"World\"!\nNew line\tTab"
}`,
			inline: false,
		},
		{
			name: "unicode characters with inline",
			json: `{"text":"こんにちは世界"}`,
			expected: `{
  "text": "こんにちは世界"
}`,
			inline: true,
		},
		{
			name: "unicode characters without inline",
			json: `{"text":"こんにちは世界"}`,
			expected: `{
  "text": "こんにちは世界"
}`,
			inline: false,
		},
		{
			name: "numbers with inline",
			json: `{"integer":42,"float":3.14159,"scientific":1.23e-4,"negative":-10}`,
			expected: `{
  "integer": 42,
  "float": 3.14159,
  "scientific": 1.23e-4,
  "negative": -10
}`,
			inline: true,
		},
		{
			name: "numbers without inline",
			json: `{"integer":42,"float":3.14159,"scientific":1.23e-4,"negative":-10}`,
			expected: `{
  "integer": 42,
  "float": 3.14159,
  "scientific": 1.23e-4,
  "negative": -10
}`,
			inline: false,
		},
		{
			name: "deeply nested structure with inline",
			json: `{"level1":{"level2":{"level3":{"level4":{"level5":"deep value"}}}}}`,
			expected: `{
  "level1": {
    "level2": {
      "level3": {
        "level4": {
          "level5": "deep value"
        }
      }
    }
  }
}`,
			inline: true,
		},
		{
			name: "deeply nested structure without inline",
			json: `{"level1":{"level2":{"level3":{"level4":{"level5":"deep value"}}}}}`,
			expected: `{
  "level1": {
    "level2": {
      "level3": {
        "level4": {
          "level5": "deep value"
        }
      }
    }
  }
}`,
			inline: false,
		},
		{
			name: "mixed array elements with inline",
			json: `{"mixed":[1,"string",true,null,{"key":"value"},[1,2,3]]}`,
			expected: `{
  "mixed": [
    1,
    "string",
    true,
    null,
    {
      "key": "value"
    },
    [
      1,
      2,
      3
    ]
  ]
}`,
			inline: true,
		},
		{
			name: "mixed array elements without inline",
			json: `{"mixed":[1,"string",true,null,{"key":"value"},[1,2,3]]}`,
			expected: `{
  "mixed": [
    1,
    "string",
    true,
    null,
    {
      "key": "value"
    },
    [
      1,
      2,
      3
    ]
  ]
}`,
			inline: false,
		},
		{
			name: "table-like structure with inline",
			json: `{"table":[[1,2,3],[4,5,6],[7,8,9]]}`,
			expected: `{
  "table": [
    [ 1, 2, 3 ],
    [ 4, 5, 6 ],
    [ 7, 8, 9 ]
  ]
}`,
			inline: true,
		},
		{
			name: "table-like structure without inline",
			json: `{"table":[[1,2,3],[4,5,6],[7,8,9]]}`,
			expected: `{
  "table": [
    [
      1,
      2,
      3
    ],
    [
      4,
      5,
      6
    ],
    [
      7,
      8,
      9
    ]
  ]
}`,
			inline: false,
		},
		{
			name: "empty string with inline",
			json: `{"value":""}`,
			expected: `{
  "value": ""
}`,
			inline: true,
		},
		{
			name: "empty string without inline",
			json: `{"value":""}`,
			expected: `{
  "value": ""
}`,
			inline: false,
		},
		{
			name: "very long string with inline",
			json: `{"longText":"This is a very long string that should not be inlined because it exceeds the maximum length for inlining. It should be displayed on its own line even when inlining is enabled."}`,
			expected: `{
  "longText": "This is a very long string that should not be inlined because it exceeds the maximum length for inlining. It should be displayed on its own line even when inlining is enabled."
}`,
			inline: true,
		},
		{
			name: "very long string without inline",
			json: `{"longText":"This is a very long string that should not be inlined because it exceeds the maximum length for inlining. It should be displayed on its own line even when inlining is enabled."}`,
			expected: `{
  "longText": "This is a very long string that should not be inlined because it exceeds the maximum length for inlining. It should be displayed on its own line even when inlining is enabled."
}`,
			inline: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := jsonx.Parse([]byte(tt.json))
			require.NoError(t, err)

			output := pretty.Print(node, tt.inline)
			strippedOutput := stripEscapeSequences(output)
			assert.Equal(t, tt.expected, strippedOutput,
				"Output doesn't match expected for %s", tt.name)
		})
	}
}

func TestPrettyPrintEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected string
		inline   bool
	}{
		{
			name: "very large array with inline",
			json: `{"largeArray":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30]}`,
			expected: `{
  "largeArray": [ 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30 ]
}`,
			inline: true,
		},
		{
			name: "very large array without inline",
			json: `{"largeArray":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30]}`,
			expected: `{
  "largeArray": [
    1,
    2,
    3,
    4,
    5,
    6,
    7,
    8,
    9,
    10,
    11,
    12,
    13,
    14,
    15,
    16,
    17,
    18,
    19,
    20,
    21,
    22,
    23,
    24,
    25,
    26,
    27,
    28,
    29,
    30
  ]
}`,
			inline: false,
		},
		{
			name: "object with many properties with inline",
			json: `{"prop1":"value1","prop2":"value2","prop3":"value3","prop4":"value4","prop5":"value5","prop6":"value6","prop7":"value7","prop8":"value8","prop9":"value9","prop10":"value10"}`,
			expected: `{
  "prop1": "value1",
  "prop2": "value2",
  "prop3": "value3",
  "prop4": "value4",
  "prop5": "value5",
  "prop6": "value6",
  "prop7": "value7",
  "prop8": "value8",
  "prop9": "value9",
  "prop10": "value10"
}`,
			inline: true,
		},
		{
			name: "object with many properties without inline",
			json: `{"prop1":"value1","prop2":"value2","prop3":"value3","prop4":"value4","prop5":"value5","prop6":"value6","prop7":"value7","prop8":"value8","prop9":"value9","prop10":"value10"}`,
			expected: `{
  "prop1": "value1",
  "prop2": "value2",
  "prop3": "value3",
  "prop4": "value4",
  "prop5": "value5",
  "prop6": "value6",
  "prop7": "value7",
  "prop8": "value8",
  "prop9": "value9",
  "prop10": "value10"
}`,
			inline: false,
		},
		{
			name: "special number values with inline",
			json: `{"nan":NaN,"infinity":Infinity,"negInfinity":-Infinity}`,
			expected: `{
  "nan": NaN,
  "infinity": Infinity,
  "negInfinity": -Infinity
}`,
			inline: true,
		},
		{
			name: "special number values without inline",
			json: `{"nan":NaN,"infinity":Infinity,"negInfinity":-Infinity}`,
			expected: `{
  "nan": NaN,
  "infinity": Infinity,
  "negInfinity": -Infinity
}`,
			inline: false,
		},
		{
			name: "object with numeric keys with inline",
			json: `{"1":"one","2":"two","3":"three"}`,
			expected: `{
  "1": "one",
  "2": "two",
  "3": "three"
}`,
			inline: true,
		},
		{
			name: "object with numeric keys without inline",
			json: `{"1":"one","2":"two","3":"three"}`,
			expected: `{
  "1": "one",
  "2": "two",
  "3": "three"
}`,
			inline: false,
		},
		{
			name: "object with empty keys with inline",
			json: `{"":"empty key"}`,
			expected: `{
  "": "empty key"
}`,
			inline: true,
		},
		{
			name: "object with empty keys without inline",
			json: `{"":"empty key"}`,
			expected: `{
  "": "empty key"
}`,
			inline: false,
		},
		{
			name: "array with single element with inline",
			json: `{"singleElement":[42]}`,
			expected: `{
  "singleElement": [ 42 ]
}`,
			inline: true,
		},
		{
			name: "array with single element without inline",
			json: `{"singleElement":[42]}`,
			expected: `{
  "singleElement": [
    42
  ]
}`,
			inline: false,
		},
		{
			name: "nested empty structures with inline",
			json: `{"emptyObject":{},"emptyArray":[],"nestedEmpty":{"empty":{},"alsoEmpty":[]}}`,
			expected: `{
  "emptyObject": {},
  "emptyArray": [],
  "nestedEmpty": {
    "empty": {},
    "alsoEmpty": []
  }
}`,
			inline: true,
		},
		{
			name: "nested empty structures without inline",
			json: `{"emptyObject":{},"emptyArray":[],"nestedEmpty":{"empty":{},"alsoEmpty":[]}}`,
			expected: `{
  "emptyObject": {},
  "emptyArray": [],
  "nestedEmpty": {
    "empty": {},
    "alsoEmpty": []
  }
}`,
			inline: false,
		},
		{
			name: "escaped characters in keys with inline",
			json: `{"escaped\\key":"value","key\\nwith\\tnewline":"value"}`,
			expected: `{
  "escaped\\key": "value",
  "key\\nwith\\tnewline": "value"
}`,
			inline: true,
		},
		{
			name: "escaped characters in keys without inline",
			json: `{"escaped\\key":"value","key\\nwith\\tnewline":"value"}`,
			expected: `{
  "escaped\\key": "value",
  "key\\nwith\\tnewline": "value"
}`,
			inline: false,
		},
		{
			name: "zero-width characters with inline",
			json: `{"zeroWidth":"invisible\u200Bjoiner\u200Chere"}`,
			expected: `{
  "zeroWidth": "invisible\u200Bjoiner\u200Chere"
}`,
			inline: true,
		},
		{
			name: "zero-width characters without inline",
			json: `{"zeroWidth":"invisible\u200Bjoiner\u200Chere"}`,
			expected: `{
  "zeroWidth": "invisible\u200Bjoiner\u200Chere"
}`,
			inline: false,
		},
		{
			name: "extremely deep nesting without inline",
			json: `{"l1":{"l2":{"l3":{"l4":{"l5":{"l6":{"l7":{"l8":{"l9":{"l10":{"l11":{"l12":{"l13":{"l14":{"l15":{"l16":{"l17":{"l18":{"l19":{"l20":"deep"}}}}}}}}}}}}}}}}}}}}`,
			expected: `{
  "l1": {
    "l2": {
      "l3": {
        "l4": {
          "l5": {
            "l6": {
              "l7": {
                "l8": {
                  "l9": {
                    "l10": {
                      "l11": {
                        "l12": {
                          "l13": {
                            "l14": {
                              "l15": {
                                "l16": {
                                  "l17": {
                                    "l18": {
                                      "l19": {
                                        "l20": "deep"
                                      }
                                    }
                                  }
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}`,
			inline: false,
		},
		{
			name: "unusual whitespace with inline",
			json: `{   "spaces"   :  "many   spaces"  ,  "tabs"  :  "tabs\ttabs\t"  }`,
			expected: `{
  "spaces": "many   spaces",
  "tabs": "tabs\ttabs\t"
}`,
			inline: true,
		},
		{
			name: "unusual whitespace without inline",
			json: `{   "spaces"   :  "many   spaces"  ,  "tabs"  :  "tabs\ttabs\t"  }`,
			expected: `{
  "spaces": "many   spaces",
  "tabs": "tabs\ttabs\t"
}`,
			inline: false,
		},
		{
			name: "extremely long key name with inline",
			json: `{"thisIsAnExtremelyLongKeyNameThatShouldTestTheFormattingCapabilitiesOfThePrettyPrinterAndEnsureThatItHandlesVeryLongKeysCorrectlyWithoutBreakingOrCausingAnyIssuesInTheOutput":"value"}`,
			expected: `{
  "thisIsAnExtremelyLongKeyNameThatShouldTestTheFormattingCapabilitiesOfThePrettyPrinterAndEnsureThatItHandlesVeryLongKeysCorrectlyWithoutBreakingOrCausingAnyIssuesInTheOutput": "value"
}`,
			inline: true,
		},
		{
			name: "extremely long key name without inline",
			json: `{"thisIsAnExtremelyLongKeyNameThatShouldTestTheFormattingCapabilitiesOfThePrettyPrinterAndEnsureThatItHandlesVeryLongKeysCorrectlyWithoutBreakingOrCausingAnyIssuesInTheOutput":"value"}`,
			expected: `{
  "thisIsAnExtremelyLongKeyNameThatShouldTestTheFormattingCapabilitiesOfThePrettyPrinterAndEnsureThatItHandlesVeryLongKeysCorrectlyWithoutBreakingOrCausingAnyIssuesInTheOutput": "value"
}`,
			inline: false,
		},
		{
			name: "unusual escape sequences with inline",
			json: `{"escapes":"\\u0000\\u0001\\u0002\\u0003\\u0004\\u0005\\u0006\\u0007\\b\\t\\n\\u000B\\f\\r\\u000E\\u000F"}`,
			expected: `{
  "escapes": "\\u0000\\u0001\\u0002\\u0003\\u0004\\u0005\\u0006\\u0007\\b\\t\\n\\u000B\\f\\r\\u000E\\u000F"
}`,
			inline: true,
		},
		{
			name: "unusual escape sequences without inline",
			json: `{"escapes":"\\u0000\\u0001\\u0002\\u0003\\u0004\\u0005\\u0006\\u0007\\b\\t\\n\\u000B\\f\\r\\u000E\\u000F"}`,
			expected: `{
  "escapes": "\\u0000\\u0001\\u0002\\u0003\\u0004\\u0005\\u0006\\u0007\\b\\t\\n\\u000B\\f\\r\\u000E\\u000F"
}`,
			inline: false,
		},
		{
			name: "array with mixed sized elements with inline",
			json: `{"mixedArray":["small",{"medium":"object with some text"},{"large":"this is a much larger object with significantly more text that should cause different formatting depending on the pretty printer settings"}]}`,
			expected: `{
  "mixedArray": [
    "small",
    {
      "medium": "object with some text"
    },
    {
      "large": "this is a much larger object with significantly more text that should cause different formatting depending on the pretty printer settings"
    }
  ]
}`,
			inline: true,
		},
		{
			name: "array with mixed sized elements without inline",
			json: `{"mixedArray":["small",{"medium":"object with some text"},{"large":"this is a much larger object with significantly more text that should cause different formatting depending on the pretty printer settings"}]}`,
			expected: `{
  "mixedArray": [
    "small",
    {
      "medium": "object with some text"
    },
    {
      "large": "this is a much larger object with significantly more text that should cause different formatting depending on the pretty printer settings"
    }
  ]
}`,
			inline: false,
		},
		{
			name: "boundary number values with inline",
			json: `{"maxInt":9007199254740991,"minInt":-9007199254740991,"smallFloat":0.0000000000000001,"largeFloat":1.7976931348623157e+308,"smallNegativeFloat":-0.0000000000000001}`,
			expected: `{
  "maxInt": 9007199254740991,
  "minInt": -9007199254740991,
  "smallFloat": 0.0000000000000001,
  "largeFloat": 1.7976931348623157e+308,
  "smallNegativeFloat": -0.0000000000000001
}`,
			inline: true,
		},
		{
			name: "boundary number values without inline",
			json: `{"maxInt":9007199254740991,"minInt":-9007199254740991,"smallFloat":0.0000000000000001,"largeFloat":1.7976931348623157e+308,"smallNegativeFloat":-0.0000000000000001}`,
			expected: `{
  "maxInt": 9007199254740991,
  "minInt": -9007199254740991,
  "smallFloat": 0.0000000000000001,
  "largeFloat": 1.7976931348623157e+308,
  "smallNegativeFloat": -0.0000000000000001
}`,
			inline: false,
		},
		{
			name: "complex nested structure without inline",
			json: `{"complex":{"arrays":[1,2,3,[4,5,[6,7,8]],"string",true,null,{"nested":"object"}],"objects":{"a":1,"b":"string","c":true,"d":null,"e":[1,2,3],"f":{"nested":"object"}}}}`,
			expected: `{
  "complex": {
    "arrays": [
      1,
      2,
      3,
      [
        4,
        5,
        [
          6,
          7,
          8
        ]
      ],
      "string",
      true,
      null,
      {
        "nested": "object"
      }
    ],
    "objects": {
      "a": 1,
      "b": "string",
      "c": true,
      "d": null,
      "e": [
        1,
        2,
        3
      ],
      "f": {
        "nested": "object"
      }
    }
  }
}`,
			inline: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := jsonx.Parse([]byte(tt.json))
			require.NoError(t, err)

			output := pretty.Print(node, tt.inline)
			strippedOutput := stripEscapeSequences(output)
			assert.Equal(t, tt.expected, strippedOutput,
				"Output doesn't match expected for %s", tt.name)
		})
	}
}
