package random

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	MRAND "math/rand"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestStringGenerator_Generate_successful(t *testing.T) {
	type testCase struct {
		timeout   time.Duration
		generator *StringGenerator
	}

	tests := map[string]testCase{
		"common rules": {
			timeout: 1 * time.Second,
			generator: &StringGenerator{
				Length: 20,
				Rules: []Rule{
					Charset{
						Charset:  LowercaseRuneset,
						MinChars: 1,
					},
					Charset{
						Charset:  UppercaseRuneset,
						MinChars: 1,
					},
					Charset{
						Charset:  NumericRuneset,
						MinChars: 1,
					},
					Charset{
						Charset:  ShortSymbolRuneset,
						MinChars: 1,
					},
				},
				charset: AlphaNumericShortSymbolRuneset,
			},
		},
		"charset not explicitly specified": {
			timeout: 1 * time.Second,
			generator: &StringGenerator{
				Length: 20,
				Rules: []Rule{
					Charset{
						Charset:  LowercaseRuneset,
						MinChars: 1,
					},
					Charset{
						Charset:  UppercaseRuneset,
						MinChars: 1,
					},
					Charset{
						Charset:  NumericRuneset,
						MinChars: 1,
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// One context to rule them all, one context to find them, one context to bring them all and in the darkness bind them.
			ctx, cancel := context.WithTimeout(context.Background(), test.timeout)
			defer cancel()

			runeset := map[rune]bool{}
			runesFound := []rune{}

			for i := 0; i < 100; i++ {
				actual, err := test.generator.Generate(ctx)
				if err != nil {
					t.Fatalf("no error expected, but got: %s", err)
				}
				for _, r := range actual {
					if runeset[r] {
						continue
					}
					runeset[r] = true
					runesFound = append(runesFound, r)
				}
			}

			sort.Sort(runes(runesFound))

			expectedCharset := getChars(test.generator.Rules)

			if !reflect.DeepEqual(runesFound, expectedCharset) {
				t.Fatalf("Didn't find all characters from the charset\nActual  : [%s]\nExpected: [%s]", string(runesFound), string(expectedCharset))
			}
		})
	}
}

func TestStringGenerator_Generate_errors(t *testing.T) {
	type testCase struct {
		timeout   time.Duration
		generator *StringGenerator
	}

	tests := map[string]testCase{
		"already timed out": {
			timeout: 0,
			generator: &StringGenerator{
				Length: 20,
				Rules: []Rule{
					testCharsetRule{
						fail: false,
					},
				},
				charset: AlphaNumericShortSymbolRuneset,
				rng:     rand.Reader,
			},
		},
		"impossible rules": {
			timeout: 10 * time.Millisecond, // Keep this short so the test doesn't take too long
			generator: &StringGenerator{
				Length: 20,
				Rules: []Rule{
					testCharsetRule{
						fail: true,
					},
				},
				charset: AlphaNumericShortSymbolRuneset,
				rng:     rand.Reader,
			},
		},
		"bad RNG reader": {
			timeout: 10 * time.Millisecond, // Keep this short so the test doesn't take too long
			generator: &StringGenerator{
				Length:  20,
				Rules:   []Rule{},
				charset: AlphaNumericShortSymbolRuneset,
				rng:     badReader{},
			},
		},
		"0 length": {
			timeout: 10 * time.Millisecond,
			generator: &StringGenerator{
				Length: 0,
				Rules: []Rule{
					Charset{
						Charset:  []rune("abcde"),
						MinChars: 0,
					},
				},
				charset: []rune("abcde"),
				rng:     rand.Reader,
			},
		},
		"-1 length": {
			timeout: 10 * time.Millisecond,
			generator: &StringGenerator{
				Length: -1,
				Rules: []Rule{
					Charset{
						Charset:  []rune("abcde"),
						MinChars: 0,
					},
				},
				charset: []rune("abcde"),
				rng:     rand.Reader,
			},
		},
		"no charset": {
			timeout: 10 * time.Millisecond,
			generator: &StringGenerator{
				Length: 20,
				Rules:  []Rule{},
				rng:    rand.Reader,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// sg := StringGenerator{
			// 	Length:  20,
			// 	charset: []rune(test.charset),
			// 	Rules:   test.rules,
			// 	rng:     test.rng,
			// }

			// One context to rule them all, one context to find them, one context to bring them all and in the darkness bind them.
			ctx, cancel := context.WithTimeout(context.Background(), test.timeout)
			defer cancel()

			actual, err := test.generator.Generate(ctx)
			if err == nil {
				t.Fatalf("Expected error but none found")
			}
			if actual != "" {
				t.Fatalf("Random string returned: %s", actual)
			}
		})
	}
}

func TestRandomRunes_deterministic(t *testing.T) {
	// These tests are to ensure that the charset selection doesn't do anything weird like selecting the same character
	// over and over again. The number of test cases here should be kept to a minimum since they are sensitive to changes
	type testCase struct {
		rngSeed  int64
		charset  string
		length   int
		expected string
	}

	tests := map[string]testCase{
		"small charset": {
			rngSeed:  1585593298447807000,
			charset:  "abcde",
			length:   20,
			expected: "ddddddcdebbeebdbdbcd",
		},
		"common charset": {
			rngSeed:  1585593298447807001,
			charset:  AlphaNumericShortSymbolCharset,
			length:   20,
			expected: "1ON6lVjnBs84zJbUBVEz",
		},
		"max size charset": {
			rngSeed: 1585593298447807002,
			charset: " !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_" +
				"`abcdefghijklmnopqrstuvwxyz{|}~ĀāĂăĄąĆćĈĉĊċČčĎďĐđĒēĔĕĖėĘęĚěĜĝĞğĠ" +
				"ġĢģĤĥĦħĨĩĪīĬĭĮįİıĲĳĴĵĶķĸĹĺĻļĽľĿŀŁłŃńŅņŇňŉŊŋŌōŎŏŐőŒœŔŕŖŗŘřŚśŜŝŞşŠ" +
				"šŢţŤťŦŧŨũŪūŬŭŮůŰűŲųŴŵŶŷŸŹźŻżŽžſ℀℁ℂ℃℄℅℆ℇ℈℉ℊℋℌℍℎℏℐℑℒℓ℔ℕ№℗℘ℙℚℛℜℝ℞℟",
			length:   20,
			expected: "tųŎ℄ņ℃Œ.@řHš-ℍ}ħGĲLℏ",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rng := MRAND.New(MRAND.NewSource(test.rngSeed))
			runes, err := randomRunes(rng, []rune(test.charset), test.length)
			if err != nil {
				t.Fatalf("Expected no error, but found: %s", err)
			}

			str := string(runes)

			if str != test.expected {
				t.Fatalf("Actual: %s  Expected: %s", str, test.expected)
			}
		})
	}
}

func TestRandomRunes_successful(t *testing.T) {
	type testCase struct {
		charset []rune // Assumes no duplicate runes
		length  int
	}

	tests := map[string]testCase{
		"small charset": {
			charset: []rune("abcde"),
			length:  20,
		},
		"common charset": {
			charset: AlphaNumericShortSymbolRuneset,
			length:  20,
		},
		"max size charset": {
			charset: []rune(
				" !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_" +
					"`abcdefghijklmnopqrstuvwxyz{|}~ĀāĂăĄąĆćĈĉĊċČčĎďĐđĒēĔĕĖėĘęĚěĜĝĞğĠ" +
					"ġĢģĤĥĦħĨĩĪīĬĭĮįİıĲĳĴĵĶķĸĹĺĻļĽľĿŀŁłŃńŅņŇňŉŊŋŌōŎŏŐőŒœŔŕŖŗŘřŚśŜŝŞşŠ" +
					"šŢţŤťŦŧŨũŪūŬŭŮůŰűŲųŴŵŶŷŸŹźŻżŽžſ℀℁ℂ℃℄℅℆ℇ℈℉ℊℋℌℍℎℏℐℑℒℓ℔ℕ№℗℘ℙℚℛℜℝ℞℟",
			),
			length: 20,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			runeset := map[rune]bool{}
			runesFound := []rune{}

			for i := 0; i < 10000; i++ {
				actual, err := randomRunes(rand.Reader, test.charset, test.length)
				if err != nil {
					t.Fatalf("no error expected, but got: %s", err)
				}
				for _, r := range actual {
					if runeset[r] {
						continue
					}
					runeset[r] = true
					runesFound = append(runesFound, r)
				}
			}

			sort.Sort(runes(runesFound))

			// Sort the input too just to ensure that they can be compared
			sort.Sort(runes(test.charset))

			if !reflect.DeepEqual(runesFound, test.charset) {
				t.Fatalf("Didn't find all characters from the charset\nActual  : [%s]\nExpected: [%s]", string(runesFound), string(test.charset))
			}
		})
	}
}

func TestRandomRunes_errors(t *testing.T) {
	type testCase struct {
		charset []rune
		length  int
		rng     io.Reader
	}

	tests := map[string]testCase{
		"nil charset": {
			charset: nil,
			length:  20,
			rng:     rand.Reader,
		},
		"empty charset": {
			charset: []rune{},
			length:  20,
			rng:     rand.Reader,
		},
		"charset is too long": {
			charset: []rune(" !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_" +
				"`abcdefghijklmnopqrstuvwxyz{|}~ĀāĂăĄąĆćĈĉĊċČčĎďĐđĒēĔĕĖėĘęĚěĜĝĞğĠ" +
				"ġĢģĤĥĦħĨĩĪīĬĭĮįİıĲĳĴĵĶķĸĹĺĻļĽľĿŀŁłŃńŅņŇňŉŊŋŌōŎŏŐőŒœŔŕŖŗŘřŚśŜŝŞşŠ" +
				"šŢţŤťŦŧŨũŪūŬŭŮůŰűŲųŴŵŶŷŸŹźŻżŽžſ℀℁ℂ℃℄℅℆ℇ℈℉ℊℋℌℍℎℏℐℑℒℓ℔ℕ№℗℘ℙℚℛℜℝ℞℟" +
				"Σ",
			),
			rng: rand.Reader,
		},
		"length is zero": {
			charset: []rune("abcde"),
			length:  0,
			rng:     rand.Reader,
		},
		"length is negative": {
			charset: []rune("abcde"),
			length:  -3,
			rng:     rand.Reader,
		},
		"reader failed": {
			charset: []rune("abcde"),
			length:  20,
			rng:     badReader{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual, err := randomRunes(test.rng, test.charset, test.length)
			if err == nil {
				t.Fatalf("Expected error but none found")
			}
			if actual != nil {
				t.Fatalf("Expected no value, but found [%s]", string(actual))
			}
		})
	}
}

func BenchmarkStringGenerator_Generate(b *testing.B) {
	lengths := []int{
		8, 12, 16, 20, 24, 28,
	}

	type testCase struct {
		generator StringGenerator
	}

	benches := map[string]testCase{
		"no rules": {
			generator: StringGenerator{
				charset: AlphaNumericFullSymbolRuneset,
				Rules:   []Rule{},
			},
		},
		"default generator": {
			generator: DefaultStringGenerator,
		},
		"large symbol set": {
			generator: StringGenerator{
				charset: AlphaNumericFullSymbolRuneset,
				Rules: []Rule{
					Charset{
						Charset:  LowercaseRuneset,
						MinChars: 1,
					},
					Charset{
						Charset:  UppercaseRuneset,
						MinChars: 1,
					},
					Charset{
						Charset:  NumericRuneset,
						MinChars: 1,
					},
					Charset{
						Charset:  FullSymbolRuneset,
						MinChars: 1,
					},
				},
			},
		},
		"max symbol set": {
			generator: StringGenerator{
				charset: []rune(" !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_" +
					"`abcdefghijklmnopqrstuvwxyz{|}~ĀāĂăĄąĆćĈĉĊċČčĎďĐđĒēĔĕĖėĘęĚěĜĝĞğĠ" +
					"ġĢģĤĥĦħĨĩĪīĬĭĮįİıĲĳĴĵĶķĸĹĺĻļĽľĿŀŁłŃńŅņŇňŉŊŋŌōŎŏŐőŒœŔŕŖŗŘřŚśŜŝŞşŠ" +
					"šŢţŤťŦŧŨũŪūŬŭŮůŰűŲųŴŵŶŷŸŹźŻżŽžſ℀℁ℂ℃℄℅℆ℇ℈℉ℊℋℌℍℎℏℐℑℒℓ℔ℕ№℗℘ℙℚℛℜℝ℞℟",
				),
				Rules: []Rule{
					Charset{
						Charset:  LowercaseRuneset,
						MinChars: 1,
					},
					Charset{
						Charset:  UppercaseRuneset,
						MinChars: 1,
					},
					Charset{
						Charset:  []rune("ĩĪīĬĭĮįİıĲĳĴĵĶķĸĹĺĻļĽľĿŀŁłŃńŅņŇňŉŊŋŌōŎŏŐőŒ"),
						MinChars: 1,
					},
				},
			},
		},
		"restrictive charset rules": {
			generator: StringGenerator{
				charset: AlphaNumericShortSymbolRuneset,
				Rules: []Rule{
					Charset{
						Charset:  []rune("A"),
						MinChars: 1,
					},
					Charset{
						Charset:  []rune("1"),
						MinChars: 1,
					},
					Charset{
						Charset:  []rune("a"),
						MinChars: 1,
					},
					Charset{
						Charset:  []rune("-"),
						MinChars: 1,
					},
				},
			},
		},
	}

	for name, bench := range benches {
		b.Run(name, func(b *testing.B) {
			for _, length := range lengths {
				bench.generator.Length = length
				b.Run(fmt.Sprintf("length=%d", length), func(b *testing.B) {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()

					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						str, err := bench.generator.Generate(ctx)
						if err != nil {
							b.Fatalf("Failed to generate string: %s", err)
						}
						if str == "" {
							b.Fatalf("Didn't error but didn't generate a string")
						}
					}
				})
			}
		})
	}

	// Mimic what the SQLCredentialsProducer is doing
	b.Run("SQLCredentialsProducer", func(b *testing.B) {
		sg := StringGenerator{
			Length:  16, // 16 because the SQLCredentialsProducer prepends 4 characters to a 20 character password
			charset: AlphaNumericRuneset,
			Rules:   nil,
			rng:     rand.Reader,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			str, err := sg.Generate(ctx)
			if err != nil {
				b.Fatalf("Failed to generate string: %s", err)
			}
			if str == "" {
				b.Fatalf("Didn't error but didn't generate a string")
			}
		}
	})
}

// Ensure the StringGenerator can be properly JSON-ified
func TestStringGenerator_JSON(t *testing.T) {
	expected := StringGenerator{
		Length:  20,
		charset: deduplicateRunes([]rune("teststring" + ShortSymbolCharset)),
		Rules: []Rule{
			testCharsetRule{
				String:  "teststring",
				Integer: 123,
			},
			Charset{
				Charset:  ShortSymbolRuneset,
				MinChars: 1,
			},
		},
	}

	b, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %s", err)
	}

	parser := Parser{
		RuleRegistry: Registry{
			Rules: map[string]ruleConstructor{
				"testrule": newTestRule,
				"charset":  ParseCharset,
			},
		},
	}
	actual, err := parser.Parse(string(b))
	if err != nil {
		t.Fatalf("Failed to parse JSON: %s", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Actual: %#v\nExpected: %#v", actual, expected)
	}
}

type badReader struct{}

func (badReader) Read([]byte) (int, error) {
	return 0, fmt.Errorf("test error")
}

func TestValidate(t *testing.T) {
	type testCase struct {
		generator StringGenerator
		expectErr bool
	}

	tests := map[string]testCase{
		"default generator": {
			generator: DefaultStringGenerator,
			expectErr: false,
		},
		"length is 0": {
			generator: StringGenerator{
				Length: 0,
			},
			expectErr: true,
		},
		"length is negative": {
			generator: StringGenerator{
				Length: -2,
			},
			expectErr: true,
		},
		"nil charset, no rules": {
			generator: StringGenerator{
				Length:  5,
				charset: nil,
			},
			expectErr: true,
		},
		"zero length charset, no rules": {
			generator: StringGenerator{
				Length:  5,
				charset: []rune{},
			},
			expectErr: true,
		},
		"rules require password longer than length": {
			generator: StringGenerator{
				Length:  5,
				charset: []rune("abcde"),
				Rules: []Rule{
					Charset{
						Charset:  []rune("abcde"),
						MinChars: 6,
					},
				},
			},
			expectErr: true,
		},
		"charset has non-printable characters": {
			generator: StringGenerator{
				Length: 0,
				charset: []rune{
					'a',
					'b',
					0, // Null character
					'd',
					'e',
				},
			},
			expectErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := test.generator.validateConfig()
			if test.expectErr && err == nil {
				t.Fatalf("err expected, got nil")
			}
			if !test.expectErr && err != nil {
				t.Fatalf("no error expected, got: %s", err)
			}
		})
	}
}

type testNonCharsetRule struct {
	String string `mapstructure:"string" json:"string"`
}

func (tr testNonCharsetRule) Pass([]rune) bool { return true }
func (tr testNonCharsetRule) Type() string     { return "testNonCharsetRule" }

func TestGetChars(t *testing.T) {
	type testCase struct {
		rules    []Rule
		expected []rune
	}

	tests := map[string]testCase{
		"nil rules": {
			rules:    nil,
			expected: []rune(nil),
		},
		"empty rules": {
			rules:    []Rule{},
			expected: []rune(nil),
		},
		"rule without chars": {
			rules: []Rule{
				testNonCharsetRule{
					String: "teststring",
				},
			},
			expected: []rune(nil),
		},
		"rule with chars": {
			rules: []Rule{
				Charset{
					Charset:  []rune("abcdefghij"),
					MinChars: 1,
				},
			},
			expected: []rune("abcdefghij"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual := getChars(test.rules)
			if !reflect.DeepEqual(actual, test.expected) {
				t.Fatalf("Actual: %v\nExpected: %v", actual, test.expected)
			}
		})
	}
}

func TestDeduplicateRunes(t *testing.T) {
	type testCase struct {
		input    []rune
		expected []rune
	}

	tests := map[string]testCase{
		"empty string": {
			input:    []rune(""),
			expected: []rune(nil),
		},
		"no duplicates": {
			input:    []rune("abcde"),
			expected: []rune("abcde"),
		},
		"in order duplicates": {
			input:    []rune("aaaabbbbcccccccddddeeeee"),
			expected: []rune("abcde"),
		},
		"out of order duplicates": {
			input:    []rune("abcdeabcdeabcdeabcde"),
			expected: []rune("abcde"),
		},
		"unicode no duplicates": {
			input:    []rune("日本語"),
			expected: []rune("日本語"),
		},
		"unicode in order duplicates": {
			input:    []rune("日日日日本本本語語語語語"),
			expected: []rune("日本語"),
		},
		"unicode out of order duplicates": {
			input:    []rune("日本語日本語日本語日本語"),
			expected: []rune("日本語"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual := deduplicateRunes(test.input)
			if !reflect.DeepEqual(actual, test.expected) {
				t.Fatalf("Actual: %#v\nExpected:%#v", actual, test.expected)
			}
		})
	}
}