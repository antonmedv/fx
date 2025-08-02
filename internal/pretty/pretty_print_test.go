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

const (
	yes byte = iota
	no
	both
)

func TestPrettyPrint(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected string
		inline   byte
	}{
		{
			name:     "standalone null with inline",
			json:     `null`,
			expected: `null`,
			inline:   both,
		},
		{
			name:     "standalone true with inline",
			json:     `true`,
			expected: `true`,
			inline:   both,
		},
		{
			name:     "standalone false with inline",
			json:     `false`,
			expected: `false`,
			inline:   both,
		},
		{
			name: "array with empty object and empty array with inline",
			json: `[{}, []]`,
			expected: `[
  {},
  []
]`,
			inline: both,
		},
		{
			name: "simple object with inline",
			json: `{"name":"John","age":30,"city":"New York"}`,
			expected: `{
  "name": "John",
  "age": 30,
  "city": "New York"
}`,
			inline: both,
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
			inline: both,
		},
		{
			name: "array of numbers with inline",
			json: `{"numbers":[1,2,3,4,5]}`,
			expected: `{
  "numbers": [ 1, 2, 3, 4, 5 ]
}`,
			inline: yes,
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
			inline: no,
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
			inline: yes,
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
			inline: no,
		},
		{
			name:     "empty object with inline",
			json:     `{}`,
			expected: `{}`,
			inline:   both,
		},
		{
			name:     "empty array with inline",
			json:     `[]`,
			expected: `[]`,
			inline:   both,
		},
		{
			name: "null value with inline",
			json: `{"value":null}`,
			expected: `{
  "value": null
}`,
			inline: both,
		},
		{
			name: "boolean values with inline",
			json: `{"active":true,"verified":false}`,
			expected: `{
  "active": true,
  "verified": false
}`,
			inline: both,
		},
		{
			name: "string with special characters without inline",
			json: `{"message":"Hello, \"World\"!\nNew line\tTab"}`,
			expected: `{
  "message": "Hello, \"World\"!\nNew line\tTab"
}`,
			inline: both,
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
			inline: both,
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
			inline: both,
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
			inline: yes,
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
			inline: no,
		},
		{
			name: "empty string with inline",
			json: `{"value":""}`,
			expected: `{
  "value": ""
}`,
			inline: both,
		},
		{
			name: "very long string with inline",
			json: `{"longText":"This is a very long string that should not be inlined because it exceeds the maximum length for inlining. It should be displayed on its own line even when inlining is enabled."}`,
			expected: `{
  "longText": "This is a very long string that should not be inlined because it exceeds the maximum length for inlining. It should be displayed on its own line even when inlining is enabled."
}`,
			inline: both,
		},
		{
			name: "very large array with inline",
			json: `{"largeArray":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30]}`,
			expected: `{
  "largeArray": [ 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30 ]
}`,
			inline: yes,
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
			inline: no,
		},
		{
			name: "special number values without inline",
			json: `{"nan":NaN,"infinity":Infinity,"negInfinity":-Infinity}`,
			expected: `{
  "nan": NaN,
  "infinity": Infinity,
  "negInfinity": -Infinity
}`,
			inline: both,
		},
		{
			name: "object with empty keys without inline",
			json: `{"":"empty key"}`,
			expected: `{
  "": "empty key"
}`,
			inline: both,
		},
		{
			name: "array with single element with inline",
			json: `{"singleElement":[42]}`,
			expected: `{
  "singleElement": [ 42 ]
}`,
			inline: yes,
		},
		{
			name: "array with single element without inline",
			json: `{"singleElement":[42]}`,
			expected: `{
  "singleElement": [
    42
  ]
}`,
			inline: no,
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
			inline: both,
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
			inline: both,
		},
		{
			name: "extremely long key name without inline",
			json: `{"thisIsAnExtremelyLongKeyNameThatShouldTestTheFormattingCapabilitiesOfThePrettyPrinterAndEnsureThatItHandlesVeryLongKeysCorrectlyWithoutBreakingOrCausingAnyIssuesInTheOutput":"value"}`,
			expected: `{
  "thisIsAnExtremelyLongKeyNameThatShouldTestTheFormattingCapabilitiesOfThePrettyPrinterAndEnsureThatItHandlesVeryLongKeysCorrectlyWithoutBreakingOrCausingAnyIssuesInTheOutput": "value"
}`,
			inline: both,
		},
		{
			name: "unusual escape sequences without inline",
			json: `{"escapes":"\\u0000\\u0001\\u0002\\u0003\\u0004\\u0005\\u0006\\u0007\\b\\t\\n\\u000B\\f\\r\\u000E\\u000F"}`,
			expected: `{
  "escapes": "\\u0000\\u0001\\u0002\\u0003\\u0004\\u0005\\u0006\\u0007\\b\\t\\n\\u000B\\f\\r\\u000E\\u000F"
}`,
			inline: both,
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
			inline: both,
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
			inline: both,
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
			inline: both,
		},
		{
			name: "array with single element",
			json: `{"key":[42]}`,
			expected: `{
  "key": [ 42 ]
}`,
			inline: yes,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := jsonx.Parse([]byte(tt.json))
			require.NoError(t, err)

			if tt.inline == both {
				{
					output := pretty.Print(node, true)
					strippedOutput := stripEscapeSequences(output)
					assert.Equal(t, tt.expected, strippedOutput,
						"Output doesn't match expected for %s", tt.name)
				}
				{
					output := pretty.Print(node, false)
					strippedOutput := stripEscapeSequences(output)
					assert.Equal(t, tt.expected, strippedOutput,
						"Output doesn't match expected for %s", tt.name)
				}
			} else {
				output := pretty.Print(node, tt.inline == yes)
				strippedOutput := stripEscapeSequences(output)
				assert.Equal(t, tt.expected, strippedOutput,
					"Output doesn't match expected for %s", tt.name)
			}
		})
	}
}
