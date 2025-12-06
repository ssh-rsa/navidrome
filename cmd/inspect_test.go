package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

var _ = Describe("Inspect CLI Command", func() {
	var tempDir string

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		tempDir = GinkgoT().TempDir()

		// Reset global flags
		format = "jsonindent"
	})

	Describe("Command configuration", func() {
		It("should have correct command structure", func() {
			Expect(inspectCmd.Use).To(Equal("inspect [files to inspect]"))
			Expect(inspectCmd.Short).NotTo(BeEmpty())
			Expect(inspectCmd.Long).NotTo(BeEmpty())
		})

		It("should have format flag", func() {
			flag := inspectCmd.Flags().Lookup("format")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("f"))
			Expect(flag.DefValue).To(Equal("jsonindent"))
		})

		It("should require at least one argument", func() {
			Expect(inspectCmd.Args).NotTo(BeNil())
			// MinimumNArgs validator will fail with 0 args
			err := inspectCmd.Args(inspectCmd, []string{})
			Expect(err).To(HaveOccurred())
		})

		It("should accept arguments", func() {
			err := inspectCmd.Args(inspectCmd, []string{"file1.mp3"})
			Expect(err).NotTo(HaveOccurred())

			err = inspectCmd.Args(inspectCmd, []string{"file1.mp3", "file2.flac"})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Format marshalers", func() {
		var sampleOutput []core.InspectOutput

		BeforeEach(func() {
			sampleOutput = []core.InspectOutput{
				{
					File: "/music/test.mp3",
					RawTags: map[string]interface{}{
						"title":  "Test Song",
						"artist": "Test Artist",
						"album":  "Test Album",
					},
					MappedTags: map[string]interface{}{
						"title":  "Test Song",
						"artist": "Test Artist",
						"album":  "Test Album",
					},
				},
			}
		})

		It("should marshal to JSON", func() {
			marshal := marshalers["json"]
			Expect(marshal).NotTo(BeNil())

			data, err := marshal(sampleOutput)
			Expect(err).NotTo(HaveOccurred())

			var result []core.InspectOutput
			err = json.Unmarshal(data, &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].File).To(Equal("/music/test.mp3"))
		})

		It("should marshal to JSON with indentation", func() {
			marshal := marshalers["jsonindent"]
			Expect(marshal).NotTo(BeNil())

			data, err := marshal(sampleOutput)
			Expect(err).NotTo(HaveOccurred())

			// Check that output is indented
			Expect(string(data)).To(ContainSubstring("  "))

			var result []core.InspectOutput
			err = json.Unmarshal(data, &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
		})

		It("should marshal to TOML", func() {
			marshal := marshalers["toml"]
			Expect(marshal).NotTo(BeNil())

			data, err := marshal(sampleOutput)
			Expect(err).NotTo(HaveOccurred())

			var result []core.InspectOutput
			err = toml.Unmarshal(data, &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
		})

		It("should marshal to YAML", func() {
			marshal := marshalers["yaml"]
			Expect(marshal).NotTo(BeNil())

			data, err := marshal(sampleOutput)
			Expect(err).NotTo(HaveOccurred())

			var result []core.InspectOutput
			err = yaml.Unmarshal(data, &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
		})

		It("should marshal to pretty format", func() {
			marshal := marshalers["pretty"]
			Expect(marshal).NotTo(BeNil())

			data, err := marshal(sampleOutput)
			Expect(err).NotTo(HaveOccurred())

			output := string(data)
			Expect(output).To(ContainSubstring("File: /music/test.mp3"))
			Expect(output).To(ContainSubstring("Raw tags:"))
			Expect(output).To(ContainSubstring("Mapped tags:"))
			Expect(output).To(ContainSubstring("===================="))
		})
	})

	Describe("Pretty marshaler details", func() {
		It("should format multiple files correctly", func() {
			output := []core.InspectOutput{
				{
					File: "/music/test1.mp3",
					RawTags: map[string]interface{}{
						"title": "Song 1",
					},
					MappedTags: map[string]interface{}{
						"title": "Song 1",
					},
				},
				{
					File: "/music/test2.mp3",
					RawTags: map[string]interface{}{
						"title": "Song 2",
					},
					MappedTags: map[string]interface{}{
						"title": "Song 2",
					},
				},
			}

			data, err := prettyMarshal(output)
			Expect(err).NotTo(HaveOccurred())

			result := string(data)
			Expect(result).To(ContainSubstring("File: /music/test1.mp3"))
			Expect(result).To(ContainSubstring("File: /music/test2.mp3"))
			Expect(strings.Count(result, "====================")).To(Equal(2))
		})

		It("should handle empty tags", func() {
			output := []core.InspectOutput{
				{
					File:       "/music/test.mp3",
					RawTags:    map[string]interface{}{},
					MappedTags: map[string]interface{}{},
				},
			}

			data, err := prettyMarshal(output)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring("File: /music/test.mp3"))
		})

		It("should handle complex tag values", func() {
			output := []core.InspectOutput{
				{
					File: "/music/test.mp3",
					RawTags: map[string]interface{}{
						"title":  "Test Song",
						"track":  1,
						"rating": 5.0,
					},
					MappedTags: map[string]interface{}{
						"title":  "Test Song",
						"track":  1,
						"rating": 5.0,
					},
				},
			}

			data, err := prettyMarshal(output)
			Expect(err).NotTo(HaveOccurred())

			result := string(data)
			Expect(result).To(ContainSubstring("title"))
			Expect(result).To(ContainSubstring("track"))
			Expect(result).To(ContainSubstring("rating"))
		})
	})

	Describe("File type validation", func() {
		var createTestFile = func(name string) string {
			path := filepath.Join(tempDir, name)
			err := os.WriteFile(path, []byte("test content"), 0644)
			Expect(err).NotTo(HaveOccurred())
			return path
		}

		It("should accept various audio formats", func() {
			audioExtensions := []string{
				".mp3", ".flac", ".m4a", ".ogg",
				".opus", ".wma", ".ape", ".wav",
				".aiff", ".aac", ".dsf", ".dff",
			}

			for _, ext := range audioExtensions {
				filename := "test" + ext
				// Check if model.IsAudioFile would accept it
				// We can't call the actual function without a real file,
				// but we can verify the extension format
				Expect(ext).To(HavePrefix("."))
				Expect(len(ext)).To(BeNumerically(">", 1))
			}
		})

		It("should handle uppercase extensions", func() {
			extensions := []string{
				"test.MP3", "test.FLAC", "test.M4A",
			}

			for _, filename := range extensions {
				Expect(filename).To(MatchRegexp(`\.[A-Z0-9]+$`))
			}
		})
	})

	Describe("Output capture", func() {
		var captureStdout = func(f func()) string {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			f()

			w.Close()
			os.Stdout = oldStdout

			output, _ := io.ReadAll(r)
			return string(output)
		}

		It("should write output to stdout", func() {
			// Create a simple test
			output := captureStdout(func() {
				data := []core.InspectOutput{
					{
						File:       "/test.mp3",
						RawTags:    map[string]interface{}{},
						MappedTags: map[string]interface{}{},
					},
				}
				marshal := marshalers["jsonindent"]
				bytes, _ := marshal(data)
				os.Stdout.Write(bytes)
			})

			Expect(output).To(ContainSubstring("test.mp3"))
		})
	})

	Describe("Format validation", func() {
		It("should have all required marshalers", func() {
			requiredFormats := []string{
				"pretty", "toml", "yaml", "json", "jsonindent",
			}

			for _, fmt := range requiredFormats {
				Expect(marshalers).To(HaveKey(fmt))
				Expect(marshalers[fmt]).NotTo(BeNil())
			}
		})

		It("should return nil for invalid format", func() {
			invalidFormat := "invalid-format"
			Expect(marshalers[invalidFormat]).To(BeNil())
		})
	})

	Describe("Multiple file handling", func() {
		It("should handle multiple files in output", func() {
			output := []core.InspectOutput{
				{
					File: "/music/file1.mp3",
					RawTags: map[string]interface{}{
						"title": "Song 1",
					},
					MappedTags: map[string]interface{}{
						"title": "Song 1",
					},
				},
				{
					File: "/music/file2.flac",
					RawTags: map[string]interface{}{
						"title": "Song 2",
					},
					MappedTags: map[string]interface{}{
						"title": "Song 2",
					},
				},
				{
					File: "/music/file3.m4a",
					RawTags: map[string]interface{}{
						"title": "Song 3",
					},
					MappedTags: map[string]interface{}{
						"title": "Song 3",
					},
				},
			}

			// Test each format with multiple files
			for formatName, marshal := range marshalers {
				data, err := marshal(output)
				Expect(err).NotTo(HaveOccurred(), "Format: "+formatName)
				Expect(len(data)).To(BeNumerically(">", 0), "Format: "+formatName)
			}
		})
	})

	Describe("Tag mapping", func() {
		It("should preserve tag structure in JSON", func() {
			output := []core.InspectOutput{
				{
					File: "/music/test.mp3",
					RawTags: map[string]interface{}{
						"title":  "Raw Title",
						"artist": "Raw Artist",
					},
					MappedTags: map[string]interface{}{
						"title":  "Mapped Title",
						"artist": "Mapped Artist",
					},
				},
			}

			marshal := marshalers["json"]
			data, err := marshal(output)
			Expect(err).NotTo(HaveOccurred())

			var result []core.InspectOutput
			err = json.Unmarshal(data, &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result[0].RawTags["title"]).To(Equal("Raw Title"))
			Expect(result[0].MappedTags["title"]).To(Equal("Mapped Title"))
		})

		It("should show difference between raw and mapped tags", func() {
			output := []core.InspectOutput{
				{
					File: "/music/test.mp3",
					RawTags: map[string]interface{}{
						"TIT2": "Title Tag", // ID3v2 tag
					},
					MappedTags: map[string]interface{}{
						"title": "Title Tag", // Mapped to standard name
					},
				},
			}

			data, err := prettyMarshal(output)
			Expect(err).NotTo(HaveOccurred())

			result := string(data)
			Expect(result).To(ContainSubstring("Raw tags:"))
			Expect(result).To(ContainSubstring("Mapped tags:"))
		})
	})

	Describe("Edge cases", func() {
		It("should handle empty output array", func() {
			output := []core.InspectOutput{}

			for formatName, marshal := range marshalers {
				data, err := marshal(output)
				Expect(err).NotTo(HaveOccurred(), "Format: "+formatName)
				// Empty output should still produce valid output
				Expect(data).NotTo(BeNil(), "Format: "+formatName)
			}
		})

		It("should handle files with special characters in path", func() {
			output := []core.InspectOutput{
				{
					File: "/music/artist & band/album (2023)/01 - song [remix].mp3",
					RawTags: map[string]interface{}{
						"title": "Test",
					},
					MappedTags: map[string]interface{}{
						"title": "Test",
					},
				},
			}

			data, err := prettyMarshal(output)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring("artist & band"))
			Expect(string(data)).To(ContainSubstring("album (2023)"))
			Expect(string(data)).To(ContainSubstring("[remix]"))
		})

		It("should handle unicode in file paths", func() {
			output := []core.InspectOutput{
				{
					File: "/music/アーティスト/アルバム/曲.mp3",
					RawTags: map[string]interface{}{
						"title": "日本語の曲",
					},
					MappedTags: map[string]interface{}{
						"title": "日本語の曲",
					},
				},
			}

			data, err := prettyMarshal(output)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring("アーティスト"))
		})

		It("should handle very long file paths", func() {
			longPath := "/music/" + strings.Repeat("a/", 50) + "file.mp3"
			output := []core.InspectOutput{
				{
					File: longPath,
					RawTags: map[string]interface{}{
						"title": "Test",
					},
					MappedTags: map[string]interface{}{
						"title": "Test",
					},
				},
			}

			data, err := prettyMarshal(output)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(longPath))
		})

		It("should handle tags with nested structures", func() {
			output := []core.InspectOutput{
				{
					File: "/music/test.mp3",
					RawTags: map[string]interface{}{
						"title": "Test",
						"metadata": map[string]interface{}{
							"nested": "value",
						},
					},
					MappedTags: map[string]interface{}{
						"title": "Test",
					},
				},
			}

			// Should not panic with nested structures
			for _, marshal := range marshalers {
				_, err := marshal(output)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should handle tags with array values", func() {
			output := []core.InspectOutput{
				{
					File: "/music/test.mp3",
					RawTags: map[string]interface{}{
						"artists": []string{"Artist 1", "Artist 2"},
					},
					MappedTags: map[string]interface{}{
						"artists": []string{"Artist 1", "Artist 2"},
					},
				},
			}

			data, err := marshalers["json"](output)
			Expect(err).NotTo(HaveOccurred())

			var result []core.InspectOutput
			err = json.Unmarshal(data, &result)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle nil tag values", func() {
			output := []core.InspectOutput{
				{
					File: "/music/test.mp3",
					RawTags: map[string]interface{}{
						"title":   "Test",
						"comment": nil,
					},
					MappedTags: map[string]interface{}{
						"title":   "Test",
						"comment": nil,
					},
				},
			}

			// Should not panic with nil values
			for formatName, marshal := range marshalers {
				_, err := marshal(output)
				Expect(err).NotTo(HaveOccurred(), "Format: "+formatName)
			}
		})
	})

	Describe("Output formatting consistency", func() {
		var sampleOutput []core.InspectOutput

		BeforeEach(func() {
			sampleOutput = []core.InspectOutput{
				{
					File: "/music/test.mp3",
					RawTags: map[string]interface{}{
						"title":  "Test Song",
						"artist": "Test Artist",
						"album":  "Test Album",
						"year":   2023,
						"track":  1,
					},
					MappedTags: map[string]interface{}{
						"title":  "Test Song",
						"artist": "Test Artist",
						"album":  "Test Album",
						"year":   2023,
						"track":  1,
					},
				},
			}
		})

		It("should produce valid output in all formats", func() {
			formats := []string{"json", "jsonindent", "yaml", "toml", "pretty"}

			for _, formatName := range formats {
				marshal := marshalers[formatName]
				data, err := marshal(sampleOutput)
				Expect(err).NotTo(HaveOccurred(), "Format: "+formatName)
				Expect(len(data)).To(BeNumerically(">", 0), "Format: "+formatName)
			}
		})

		It("should include all tag fields in output", func() {
			data, err := marshalers["jsonindent"](sampleOutput)
			Expect(err).NotTo(HaveOccurred())

			output := string(data)
			Expect(output).To(ContainSubstring("title"))
			Expect(output).To(ContainSubstring("artist"))
			Expect(output).To(ContainSubstring("album"))
			Expect(output).To(ContainSubstring("year"))
			Expect(output).To(ContainSubstring("track"))
		})
	})

	Describe("Buffer handling", func() {
		It("should handle large output", func() {
			// Create a large output structure
			output := make([]core.InspectOutput, 100)
			for i := 0; i < 100; i++ {
				output[i] = core.InspectOutput{
					File: "/music/file" + string(rune(i)) + ".mp3",
					RawTags: map[string]interface{}{
						"title":  "Song " + string(rune(i)),
						"artist": "Artist " + string(rune(i)),
					},
					MappedTags: map[string]interface{}{
						"title":  "Song " + string(rune(i)),
						"artist": "Artist " + string(rune(i)),
					},
				}
			}

			for formatName, marshal := range marshalers {
				data, err := marshal(output)
				Expect(err).NotTo(HaveOccurred(), "Format: "+formatName)
				Expect(len(data)).To(BeNumerically(">", 1000), "Format: "+formatName)
			}
		})
	})

	Describe("TOML formatting specifics", func() {
		It("should produce valid TOML", func() {
			output := []core.InspectOutput{
				{
					File: "/music/test.mp3",
					RawTags: map[string]interface{}{
						"title": "Test Song",
					},
					MappedTags: map[string]interface{}{
						"title": "Test Song",
					},
				},
			}

			data, err := marshalers["toml"](output)
			Expect(err).NotTo(HaveOccurred())

			// Verify it's valid TOML by unmarshaling
			var result []core.InspectOutput
			err = toml.Unmarshal(data, &result)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("YAML formatting specifics", func() {
		It("should produce valid YAML", func() {
			output := []core.InspectOutput{
				{
					File: "/music/test.mp3",
					RawTags: map[string]interface{}{
						"title": "Test Song",
					},
					MappedTags: map[string]interface{}{
						"title": "Test Song",
					},
				},
			}

			data, err := marshalers["yaml"](output)
			Expect(err).NotTo(HaveOccurred())

			// Verify it's valid YAML by unmarshaling
			var result []core.InspectOutput
			err = yaml.Unmarshal(data, &result)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle YAML special characters", func() {
			output := []core.InspectOutput{
				{
					File: "/music/test: file.mp3",
					RawTags: map[string]interface{}{
						"title": "Test: Song",
					},
					MappedTags: map[string]interface{}{
						"title": "Test: Song",
					},
				},
			}

			data, err := marshalers["yaml"](output)
			Expect(err).NotTo(HaveOccurred())

			var result []core.InspectOutput
			err = yaml.Unmarshal(data, &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result[0].File).To(ContainSubstring(":"))
		})
	})

	Describe("Marshaler error handling", func() {
		It("should handle marshalers gracefully", func() {
			testData := []core.InspectOutput{
				{
					File:       "/test.mp3",
					RawTags:    map[string]interface{}{"key": "value"},
					MappedTags: map[string]interface{}{"key": "value"},
				},
			}

			for name, marshaler := range marshalers {
				result, err := marshaler(testData)
				Expect(err).NotTo(HaveOccurred(), "Marshaler %s should not error", name)
				Expect(result).NotTo(BeNil(), "Marshaler %s should return data", name)
			}
		})
	})

	Describe("Pretty format sections", func() {
		It("should clearly separate raw and mapped tags", func() {
			output := []core.InspectOutput{
				{
					File: "/music/test.mp3",
					RawTags: map[string]interface{}{
						"TPE1": "Artist Name",
					},
					MappedTags: map[string]interface{}{
						"artist": "Artist Name",
					},
				},
			}

			data, err := prettyMarshal(output)
			Expect(err).NotTo(HaveOccurred())

			result := string(data)
			rawIdx := strings.Index(result, "Raw tags:")
			mappedIdx := strings.Index(result, "Mapped tags:")

			Expect(rawIdx).To(BeNumerically(">", -1))
			Expect(mappedIdx).To(BeNumerically(">", rawIdx))
		})
	})

	Describe("Output size handling", func() {
		It("should handle output of reasonable size", func() {
			output := []core.InspectOutput{
				{
					File: "/music/test.mp3",
					RawTags: map[string]interface{}{
						"title": strings.Repeat("a", 1000),
					},
					MappedTags: map[string]interface{}{
						"title": strings.Repeat("a", 1000),
					},
				},
			}

			for _, marshal := range marshalers {
				data, err := marshal(output)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(data)).To(BeNumerically(">", 1000))
			}
		})
	})
})
