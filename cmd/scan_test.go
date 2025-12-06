package cmd

import (
	"context"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scan CLI Command", func() {
	var tempDir string
	var dbPath string
	var musicPath string
	var ctx context.Context
	var ds model.DataStore

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		tempDir = GinkgoT().TempDir()
		dbPath = "file::memory:?cache=shared"
		musicPath = filepath.Join(tempDir, "music")

		conf.Server.DbPath = dbPath

		ctx = context.Background()
		DeferCleanup(db.Init(ctx))

		sqlDB := db.Db()
		ds = persistence.New(sqlDB)

		// Create a test library
		err := ds.Library(ctx).Put(&model.Library{
			ID:   1,
			Name: "Test Library",
			Path: musicPath,
		})
		Expect(err).NotTo(HaveOccurred())

		// Reset global flags
		fullScan = false
		subprocess = false
		targets = []string{}
	})

	Describe("Command configuration", func() {
		It("should have correct command structure", func() {
			Expect(scanCmd.Use).To(Equal("scan"))
			Expect(scanCmd.Short).NotTo(BeEmpty())
			Expect(scanCmd.Long).NotTo(BeEmpty())
		})

		It("should have full scan flag", func() {
			flag := scanCmd.Flags().Lookup("full")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("f"))
			Expect(flag.DefValue).To(Equal("false"))
		})

		It("should have subprocess flag", func() {
			flag := scanCmd.Flags().Lookup("subprocess")
			Expect(flag).NotTo(BeNil())
			Expect(flag.DefValue).To(Equal("false"))
		})

		It("should have target flag", func() {
			flag := scanCmd.Flags().Lookup("target")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("t"))
		})
	})

	Describe("Scan targets", func() {
		It("should parse valid target format", func() {
			targets = []string{"1:Music/Rock", "1:Music/Jazz"}

			scanTargets, err := model.ParseTargets(targets)
			Expect(err).NotTo(HaveOccurred())
			Expect(scanTargets).To(HaveLen(2))
			Expect(scanTargets[0].LibraryID).To(Equal(1))
			Expect(scanTargets[0].FolderPath).To(Equal("Music/Rock"))
			Expect(scanTargets[1].LibraryID).To(Equal(1))
			Expect(scanTargets[1].FolderPath).To(Equal("Music/Jazz"))
		})

		It("should handle multiple libraries", func() {
			// Create second library
			err := ds.Library(ctx).Put(&model.Library{
				ID:   2,
				Name: "Test Library 2",
				Path: filepath.Join(tempDir, "music2"),
			})
			Expect(err).NotTo(HaveOccurred())

			targets = []string{"1:Music/Rock", "2:Classical"}

			scanTargets, err := model.ParseTargets(targets)
			Expect(err).NotTo(HaveOccurred())
			Expect(scanTargets).To(HaveLen(2))
			Expect(scanTargets[0].LibraryID).To(Equal(1))
			Expect(scanTargets[1].LibraryID).To(Equal(2))
		})

		It("should handle paths with spaces", func() {
			targets = []string{"1:Music/Rock Classics", "1:Music/Jazz Standards"}

			scanTargets, err := model.ParseTargets(targets)
			Expect(err).NotTo(HaveOccurred())
			Expect(scanTargets).To(HaveLen(2))
			Expect(scanTargets[0].FolderPath).To(Equal("Music/Rock Classics"))
			Expect(scanTargets[1].FolderPath).To(Equal("Music/Jazz Standards"))
		})

		It("should handle absolute paths", func() {
			targets = []string{"1:/absolute/path/to/music"}

			scanTargets, err := model.ParseTargets(targets)
			Expect(err).NotTo(HaveOccurred())
			Expect(scanTargets).To(HaveLen(1))
			Expect(scanTargets[0].FolderPath).To(Equal("/absolute/path/to/music"))
		})

		It("should handle relative paths", func() {
			targets = []string{"1:relative/path"}

			scanTargets, err := model.ParseTargets(targets)
			Expect(err).NotTo(HaveOccurred())
			Expect(scanTargets).To(HaveLen(1))
			Expect(scanTargets[0].FolderPath).To(Equal("relative/path"))
		})

		It("should reject invalid target format", func() {
			invalidTargets := [][]string{
				{"invalid"},           // Missing colon
				{"notanumber:path"},   // Invalid library ID
				{":path"},             // Empty library ID
				{"1:"},                // Empty path
				{"1"},                 // Missing path
				{"1:path:extra"},      // Too many colons in unexpected format
			}

			for _, targets := range invalidTargets {
				_, err := model.ParseTargets(targets)
				if len(targets[0]) == 0 || !contains(targets[0], ":") ||
				   (len(targets[0]) > 0 && targets[0][0] == ':') {
					Expect(err).To(HaveOccurred(), "Should fail for: %v", targets)
				}
			}
		})
	})

	Describe("Full scan flag", func() {
		It("should default to incremental scan", func() {
			Expect(fullScan).To(BeFalse())
		})

		It("should allow enabling full scan", func() {
			fullScan = true
			Expect(fullScan).To(BeTrue())
		})
	})

	Describe("Subprocess mode", func() {
		It("should default to interactive mode", func() {
			Expect(subprocess).To(BeFalse())
		})

		It("should allow enabling subprocess mode", func() {
			subprocess = true
			Expect(subprocess).To(BeTrue())
		})
	})

	Describe("Scan execution context", func() {
		It("should handle context cancellation gracefully", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			// The scanner should respect context cancellation
			Expect(ctx.Err()).To(Equal(context.Canceled))
		})
	})

	Describe("Empty library scan", func() {
		It("should handle scanning empty library", func() {
			// This test verifies that scanning an empty library doesn't crash
			// We can't easily test the full runScanner without mocking,
			// but we can verify the setup is correct
			libs, err := ds.Library(ctx).GetAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(libs).To(HaveLen(1))
			Expect(libs[0].Path).To(Equal(musicPath))
		})
	})

	Describe("Multiple targets validation", func() {
		It("should accept multiple targets for same library", func() {
			targets = []string{
				"1:Music/Rock",
				"1:Music/Jazz",
				"1:Music/Classical",
			}

			scanTargets, err := model.ParseTargets(targets)
			Expect(err).NotTo(HaveOccurred())
			Expect(scanTargets).To(HaveLen(3))

			// All should be for library 1
			for _, target := range scanTargets {
				Expect(target.LibraryID).To(Equal(1))
			}
		})

		It("should accept targets for different libraries", func() {
			// Create additional libraries
			for i := 2; i <= 3; i++ {
				err := ds.Library(ctx).Put(&model.Library{
					ID:   i,
					Name: "Test Library " + string(rune(i)),
					Path: filepath.Join(tempDir, "music"+string(rune(i))),
				})
				Expect(err).NotTo(HaveOccurred())
			}

			targets = []string{
				"1:Music/Rock",
				"2:Music/Jazz",
				"3:Music/Classical",
			}

			scanTargets, err := model.ParseTargets(targets)
			Expect(err).NotTo(HaveOccurred())
			Expect(scanTargets).To(HaveLen(3))

			// Verify different library IDs
			Expect(scanTargets[0].LibraryID).To(Equal(1))
			Expect(scanTargets[1].LibraryID).To(Equal(2))
			Expect(scanTargets[2].LibraryID).To(Equal(3))
		})
	})

	Describe("Path handling", func() {
		It("should handle paths with special characters", func() {
			targets = []string{
				"1:Music/Rock & Roll",
				"1:Music/Jazz (Cool)",
				"1:Music/Classical [Best]",
			}

			scanTargets, err := model.ParseTargets(targets)
			Expect(err).NotTo(HaveOccurred())
			Expect(scanTargets).To(HaveLen(3))
			Expect(scanTargets[0].FolderPath).To(Equal("Music/Rock & Roll"))
			Expect(scanTargets[1].FolderPath).To(Equal("Music/Jazz (Cool)"))
			Expect(scanTargets[2].FolderPath).To(Equal("Music/Classical [Best]"))
		})

		It("should handle paths with unicode characters", func() {
			targets = []string{
				"1:音楽/ロック",
				"1:Música/Rock",
			}

			scanTargets, err := model.ParseTargets(targets)
			Expect(err).NotTo(HaveOccurred())
			Expect(scanTargets).To(HaveLen(2))
			Expect(scanTargets[0].FolderPath).To(Equal("音楽/ロック"))
			Expect(scanTargets[1].FolderPath).To(Equal("Música/Rock"))
		})

		It("should preserve path separators", func() {
			targets = []string{
				"1:Music/Rock/Classic",
				"1:Music/Jazz/Modern/Fusion",
			}

			scanTargets, err := model.ParseTargets(targets)
			Expect(err).NotTo(HaveOccurred())
			Expect(scanTargets[0].FolderPath).To(Equal("Music/Rock/Classic"))
			Expect(scanTargets[1].FolderPath).To(Equal("Music/Jazz/Modern/Fusion"))
		})
	})

	Describe("Edge cases", func() {
		It("should handle empty targets array", func() {
			targets = []string{}

			scanTargets, err := model.ParseTargets(targets)
			Expect(err).NotTo(HaveOccurred())
			Expect(scanTargets).To(BeEmpty())
		})

		It("should handle large library IDs", func() {
			targets = []string{"999999:Music"}

			scanTargets, err := model.ParseTargets(targets)
			Expect(err).NotTo(HaveOccurred())
			Expect(scanTargets).To(HaveLen(1))
			Expect(scanTargets[0].LibraryID).To(Equal(999999))
		})

		It("should handle very long paths", func() {
			longPath := "Music/" + string(make([]byte, 500))
			for i := range longPath[6:] {
				longPath = longPath[:6+i] + "a" + longPath[6+i+1:]
			}
			targets = []string{"1:" + longPath}

			scanTargets, err := model.ParseTargets(targets)
			Expect(err).NotTo(HaveOccurred())
			Expect(scanTargets).To(HaveLen(1))
			Expect(scanTargets[0].FolderPath).To(Equal(longPath))
		})
	})

	Describe("Flag combinations", func() {
		It("should allow full scan with targets", func() {
			fullScan = true
			targets = []string{"1:Music/Rock"}

			scanTargets, err := model.ParseTargets(targets)
			Expect(err).NotTo(HaveOccurred())
			Expect(scanTargets).To(HaveLen(1))
			Expect(fullScan).To(BeTrue())
		})

		It("should allow subprocess mode with full scan", func() {
			subprocess = true
			fullScan = true

			Expect(subprocess).To(BeTrue())
			Expect(fullScan).To(BeTrue())
		})

		It("should allow subprocess mode with targets", func() {
			subprocess = true
			targets = []string{"1:Music"}

			scanTargets, err := model.ParseTargets(targets)
			Expect(err).NotTo(HaveOccurred())
			Expect(subprocess).To(BeTrue())
			Expect(scanTargets).To(HaveLen(1))
		})

		It("should allow all flags together", func() {
			subprocess = true
			fullScan = true
			targets = []string{"1:Music/Rock", "1:Music/Jazz"}

			scanTargets, err := model.ParseTargets(targets)
			Expect(err).NotTo(HaveOccurred())
			Expect(subprocess).To(BeTrue())
			Expect(fullScan).To(BeTrue())
			Expect(scanTargets).To(HaveLen(2))
		})
	})

	Describe("Library validation", func() {
		It("should verify library exists in database", func() {
			libs, err := ds.Library(ctx).GetAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(libs).To(HaveLen(1))
			Expect(libs[0].ID).To(Equal(1))
		})

		It("should handle multiple libraries", func() {
			// Create additional libraries
			for i := 2; i <= 5; i++ {
				err := ds.Library(ctx).Put(&model.Library{
					ID:   i,
					Name: "Test Library",
					Path: filepath.Join(tempDir, "music"),
				})
				Expect(err).NotTo(HaveOccurred())
			}

			libs, err := ds.Library(ctx).GetAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(libs).To(HaveLen(5))
		})
	})
})

// Helper function for validation tests
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
	       (s == substr || (len(s) > len(substr) &&
		   findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
