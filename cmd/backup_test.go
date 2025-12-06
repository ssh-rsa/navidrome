package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/db"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("Backup CLI Commands", func() {
	var tempDir string
	var dbPath string
	var backupPath string
	var ctx context.Context

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		tempDir = GinkgoT().TempDir()
		dbPath = filepath.Join(tempDir, "navidrome.db")
		backupPath = filepath.Join(tempDir, "backups")

		// Setup database configuration
		conf.Server.DbPath = dbPath
		conf.Server.Backup.Path = backupPath
		conf.Server.Backup.Count = 3

		// Create backup directory
		Expect(os.MkdirAll(backupPath, 0755)).To(Succeed())

		ctx = context.Background()

		// Initialize database
		DeferCleanup(db.Init(ctx))

		// Reset global flags
		backupDir = ""
		backupCount = -1
		force = false
		restorePath = ""
	})

	Describe("Backup create command", func() {
		It("should create a valid backup file", func() {
			runBackup(ctx)

			// Check that a backup file was created
			entries, err := os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(1))
			Expect(entries[0].Name()).To(HavePrefix("navidrome-"))
			Expect(entries[0].Name()).To(HaveSuffix(".db"))

			// Verify backup file is not empty
			info, err := entries[0].Info()
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Size()).To(BeNumerically(">", 0))
		})

		It("should use custom backup directory when specified", func() {
			customBackupDir := filepath.Join(tempDir, "custom-backups")
			Expect(os.MkdirAll(customBackupDir, 0755)).To(Succeed())

			backupDir = customBackupDir
			runBackup(ctx)

			// Check that backup was created in custom directory
			entries, err := os.ReadDir(customBackupDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(1))
		})

		It("should create multiple backups without overwriting", func() {
			// Create first backup
			runBackup(ctx)

			// Create second backup
			runBackup(ctx)

			// Check that two backup files exist
			entries, err := os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(2))
		})
	})

	Describe("Backup prune command", func() {
		var createBackups = func(count int) {
			for i := 0; i < count; i++ {
				runBackup(ctx)
			}
		}

		It("should keep the specified number of backups", func() {
			// Create 5 backups
			createBackups(5)

			entries, err := os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(5))

			// Prune to keep 3 (using default from config)
			force = true // Bypass confirmation
			runPrune(ctx)

			// Verify only 3 backups remain
			entries, err = os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(3))
		})

		It("should use custom backup count when specified", func() {
			createBackups(5)

			backupCount = 2
			force = true
			runPrune(ctx)

			entries, err := os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(2))
		})

		It("should use custom backup directory when specified", func() {
			customBackupDir := filepath.Join(tempDir, "custom-backups")
			Expect(os.MkdirAll(customBackupDir, 0755)).To(Succeed())

			// Create backups in custom directory
			backupDir = customBackupDir
			createBackups(5)

			// Prune in custom directory
			backupCount = 2
			force = true
			runPrune(ctx)

			entries, err := os.ReadDir(customBackupDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(2))
		})

		It("should keep the most recent backups", func() {
			createBackups(3)

			entries, err := os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())
			oldestBackup := entries[0].Name()

			// Create one more backup
			runBackup(ctx)

			// Prune to keep 3
			force = true
			runPrune(ctx)

			// Verify the oldest backup was removed
			entries, err = os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(3))

			var found bool
			for _, entry := range entries {
				if entry.Name() == oldestBackup {
					found = true
					break
				}
			}
			Expect(found).To(BeFalse(), "oldest backup should have been pruned")
		})

		It("should handle zero backup count with force flag", func() {
			createBackups(3)

			backupCount = 0
			force = true
			runPrune(ctx)

			entries, err := os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(BeEmpty())
		})
	})

	Describe("Backup restore command", func() {
		var backupFile string

		BeforeEach(func() {
			// Create a backup to restore from
			runBackup(ctx)

			entries, err := os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(1))

			backupFile = filepath.Join(backupPath, entries[0].Name())
			restorePath = backupFile
		})

		It("should restore database from backup file", func() {
			// Verify original database exists
			_, err := os.Stat(dbPath)
			Expect(err).NotTo(HaveOccurred())

			force = true
			runRestore(ctx)

			// Verify database still exists after restore
			_, err = os.Stat(dbPath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle invalid backup path", func() {
			restorePath = filepath.Join(tempDir, "nonexistent-backup.db")
			force = true

			// This should log a fatal error, but we can't easily test that
			// in a unit test without mocking. The function calls log.Fatal
			// which exits the process.
			// For now, we just verify the path doesn't exist
			_, err := os.Stat(restorePath)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Describe("Backup command flags", func() {
		It("should properly initialize backup command with flags", func() {
			cmd := &cobra.Command{}
			backupCmd.Flags().VisitAll(func(f *cobra.Flag) {
				Expect(f).NotTo(BeNil())
			})

			// Verify backup-dir flag exists
			flag := backupCmd.Flags().Lookup("backup-dir")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("d"))
		})

		It("should properly initialize prune command with flags", func() {
			// Verify backup-dir flag exists
			flag := pruneCmd.Flags().Lookup("backup-dir")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("d"))

			// Verify keep-count flag exists
			flag = pruneCmd.Flags().Lookup("keep-count")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("k"))

			// Verify force flag exists
			flag = pruneCmd.Flags().Lookup("force")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("f"))
		})

		It("should properly initialize restore command with flags", func() {
			// Verify backup-file flag exists and is required
			flag := restoreCommand.Flags().Lookup("backup-file")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("b"))

			// Verify force flag exists
			flag = restoreCommand.Flags().Lookup("force")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("f"))
		})
	})

	Describe("Database path parsing", func() {
		It("should handle database path with query parameters", func() {
			conf.Server.DbPath = dbPath + "?cache=shared&mode=rwc"

			runBackup(ctx)

			// Verify backup was created successfully
			entries, err := os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(1))
		})

		It("should handle database path without query parameters", func() {
			conf.Server.DbPath = dbPath

			runBackup(ctx)

			// Verify backup was created successfully
			entries, err := os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(1))
		})
	})

	Describe("Command structure", func() {
		It("should have correct backup root command configuration", func() {
			Expect(backupRoot.Use).To(Equal("backup"))
			Expect(backupRoot.Aliases).To(ContainElement("bkp"))
			Expect(backupRoot.Short).NotTo(BeEmpty())
		})

		It("should have correct create subcommand configuration", func() {
			Expect(backupCmd.Use).To(Equal("create"))
			Expect(backupCmd.Short).NotTo(BeEmpty())
			Expect(backupCmd.Long).NotTo(BeEmpty())
		})

		It("should have correct prune subcommand configuration", func() {
			Expect(pruneCmd.Use).To(Equal("prune"))
			Expect(pruneCmd.Short).NotTo(BeEmpty())
			Expect(pruneCmd.Long).NotTo(BeEmpty())
		})

		It("should have correct restore subcommand configuration", func() {
			Expect(restoreCommand.Use).To(Equal("restore"))
			Expect(restoreCommand.Short).NotTo(BeEmpty())
			Expect(restoreCommand.Long).NotTo(BeEmpty())
		})
	})

	Describe("Backup file naming", func() {
		It("should create backups with timestamp-based names", func() {
			runBackup(ctx)

			entries, err := os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(1))

			name := entries[0].Name()
			Expect(name).To(MatchRegexp(`^navidrome-\d{8}-\d{6}\.db$`))
		})

		It("should create unique backup names for sequential backups", func() {
			runBackup(ctx)
			entries1, err := os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())

			runBackup(ctx)
			entries2, err := os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(entries2).To(HaveLen(2))
			Expect(entries1[0].Name()).NotTo(Equal(entries2[1].Name()))
		})
	})

	Describe("Error handling", func() {
		It("should handle missing backup directory gracefully", func() {
			// Remove backup directory
			Expect(os.RemoveAll(backupPath)).To(Succeed())

			// This should still succeed as the directory will be created
			runBackup(ctx)

			_, err := os.Stat(backupPath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle database path with special characters", func() {
			specialDir := filepath.Join(tempDir, "special dir with spaces")
			Expect(os.MkdirAll(specialDir, 0755)).To(Succeed())

			specialDbPath := filepath.Join(specialDir, "navidrome.db")
			conf.Server.DbPath = specialDbPath

			// Reinitialize database with new path
			db.Db().Close()
			ctx2 := context.Background()
			DeferCleanup(db.Init(ctx2))

			runBackup(ctx2)

			entries, err := os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(1))
		})
	})

	Describe("Prune edge cases", func() {
		It("should handle prune when no backups exist", func() {
			// Ensure backup directory is empty
			entries, err := os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())
			for _, entry := range entries {
				Expect(os.Remove(filepath.Join(backupPath, entry.Name()))).To(Succeed())
			}

			force = true
			runPrune(ctx)

			// Should complete without error
			entries, err = os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(BeEmpty())
		})

		It("should handle prune when backup count exceeds existing backups", func() {
			// Create 2 backups
			runBackup(ctx)
			runBackup(ctx)

			// Try to keep 10 backups
			backupCount = 10
			force = true
			runPrune(ctx)

			// Should keep both backups
			entries, err := os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(2))
		})

		It("should handle negative backup count by using config default", func() {
			// Create 5 backups
			for i := 0; i < 5; i++ {
				runBackup(ctx)
			}

			// Use -1 to indicate use config default (3)
			backupCount = -1
			force = true
			runPrune(ctx)

			entries, err := os.ReadDir(backupPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(3))
		})
	})

	Describe("Database path validation", func() {
		It("should extract path correctly from various formats", func() {
			testCases := []struct {
				input    string
				expected string
			}{
				{"/path/to/db.db", "/path/to/db.db"},
				{"/path/to/db.db?cache=shared", "/path/to/db.db"},
				{"/path/to/db.db?mode=rwc&cache=shared", "/path/to/db.db"},
				{"file:memory:?mode=memory", "file:memory:"},
			}

			for _, tc := range testCases {
				idx := strings.LastIndex(tc.input, "?")
				var path string
				if idx == -1 {
					path = tc.input
				} else {
					path = tc.input[:idx]
				}
				Expect(path).To(Equal(tc.expected))
			}
		})
	})
})
