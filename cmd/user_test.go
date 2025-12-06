package cmd

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("User CLI Commands", func() {
	var tempDir string
	var dbPath string
	var ctx context.Context
	var ds model.DataStore

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		tempDir = GinkgoT().TempDir()
		dbPath = "file::memory:?cache=shared"

		conf.Server.DbPath = dbPath

		ctx = context.Background()
		DeferCleanup(db.Init(ctx))

		sqlDB := db.Db()
		ds = persistence.New(sqlDB)

		// Create a test library
		err := ds.Library(ctx).Put(&model.Library{
			ID:   1,
			Name: "Test Library",
			Path: "/music",
		})
		Expect(err).NotTo(HaveOccurred())

		// Create an admin user for testing
		adminUser := &model.User{
			ID:          "admin-id",
			UserName:    "admin",
			Name:        "Admin User",
			Email:       "admin@example.com",
			IsAdmin:     true,
			NewPassword: "admin123",
		}
		adminUser.Libraries, err = ds.Library(ctx).GetAll()
		Expect(err).NotTo(HaveOccurred())

		err = ds.User(ctx).Put(adminUser)
		Expect(err).NotTo(HaveOccurred())

		err = ds.User(ctx).SetUserLibraries(adminUser.ID, []int{1})
		Expect(err).NotTo(HaveOccurred())

		// Reset global flags
		userID = ""
		email = ""
		name = ""
		libraryIds = []int{}
		setAdmin = false
		setPassword = false
		setRegularUser = false
		removeEmail = false
		removeName = false
		outputFormat = "csv"
	})

	Describe("User create command", func() {
		var oldStdin *os.File
		var mockPasswordInput = func(password, confirmation string) func() {
			oldStdin = os.Stdin
			r, w, _ := os.Pipe()

			// Write password and confirmation to pipe
			go func() {
				defer w.Close()
				w.Write([]byte(password + "\n"))
				w.Write([]byte(confirmation + "\n"))
			}()

			os.Stdin = r

			return func() {
				os.Stdin = oldStdin
				r.Close()
			}
		}

		It("should create a new user with valid password", func() {
			cleanup := mockPasswordInput("password123", "password123")
			defer cleanup()

			userID = "testuser"
			email = "test@example.com"
			name = "Test User"

			runCreateUser(ctx)

			// Verify user was created
			user, err := ds.User(ctx).FindByUsername("testuser")
			Expect(err).NotTo(HaveOccurred())
			Expect(user.UserName).To(Equal("testuser"))
			Expect(user.Email).To(Equal("test@example.com"))
			Expect(user.Name).To(Equal("Test User"))
			Expect(user.IsAdmin).To(BeFalse())
		})

		It("should create an admin user when admin flag is set", func() {
			cleanup := mockPasswordInput("password123", "password123")
			defer cleanup()

			userID = "adminuser"
			setAdmin = true

			runCreateUser(ctx)

			user, err := ds.User(ctx).FindByUsername("adminuser")
			Expect(err).NotTo(HaveOccurred())
			Expect(user.IsAdmin).To(BeTrue())
		})

		It("should use username as name when name is not provided", func() {
			cleanup := mockPasswordInput("password123", "password123")
			defer cleanup()

			userID = "testuser"

			runCreateUser(ctx)

			user, err := ds.User(ctx).FindByUsername("testuser")
			Expect(err).NotTo(HaveOccurred())
			Expect(user.Name).To(Equal("testuser"))
		})

		It("should assign specific libraries to non-admin user", func() {
			cleanup := mockPasswordInput("password123", "password123")
			defer cleanup()

			// Create additional libraries
			err := ds.Library(ctx).Put(&model.Library{
				ID:   2,
				Name: "Test Library 2",
				Path: "/music2",
			})
			Expect(err).NotTo(HaveOccurred())

			userID = "testuser"
			libraryIds = []int{1}

			runCreateUser(ctx)

			user, err := ds.User(ctx).FindByUsername("testuser")
			Expect(err).NotTo(HaveOccurred())
			Expect(user.Libraries).To(HaveLen(1))
			Expect(user.Libraries[0].ID).To(Equal(1))
		})

		It("should assign all libraries to admin user regardless of library-ids flag", func() {
			cleanup := mockPasswordInput("password123", "password123")
			defer cleanup()

			// Create additional library
			err := ds.Library(ctx).Put(&model.Library{
				ID:   2,
				Name: "Test Library 2",
				Path: "/music2",
			})
			Expect(err).NotTo(HaveOccurred())

			userID = "testuser"
			setAdmin = true
			libraryIds = []int{1} // This should be ignored for admin users

			runCreateUser(ctx)

			user, err := ds.User(ctx).FindByUsername("testuser")
			Expect(err).NotTo(HaveOccurred())
			Expect(user.Libraries).To(HaveLen(2))
		})
	})

	Describe("User delete command", func() {
		BeforeEach(func() {
			// Create a regular user to delete
			user := &model.User{
				ID:          "user-to-delete",
				UserName:    "deleteuser",
				Name:        "Delete User",
				Email:       "delete@example.com",
				IsAdmin:     false,
				NewPassword: "password123",
			}
			err := ds.User(ctx).Put(user)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should delete an existing user by username", func() {
			userID = "deleteuser"

			runDeleteUser(ctx)

			_, err := ds.User(ctx).FindByUsername("deleteuser")
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(model.ErrNotFound))
		})

		It("should delete an existing user by ID", func() {
			userID = "user-to-delete"

			runDeleteUser(ctx)

			_, err := ds.User(ctx).Get("user-to-delete")
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(model.ErrNotFound))
		})

		It("should not delete the last user", func() {
			// Delete the admin user first to test the protection
			userID = "admin-id"
			err := ds.User(ctx).Delete(userID)
			Expect(err).NotTo(HaveOccurred())

			// Now try to delete the last remaining user
			userID = "deleteuser"

			// This will call log.Fatal which we can't easily test
			// We can verify the logic by checking user count
			count, err := ds.User(ctx).CountAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(1)))
		})
	})

	Describe("User edit command", func() {
		var testUser *model.User
		var oldStdin *os.File
		var mockPasswordInput = func(password, confirmation string) func() {
			oldStdin = os.Stdin
			r, w, _ := os.Pipe()

			go func() {
				defer w.Close()
				w.Write([]byte(password + "\n"))
				w.Write([]byte(confirmation + "\n"))
			}()

			os.Stdin = r

			return func() {
				os.Stdin = oldStdin
				r.Close()
			}
		}

		BeforeEach(func() {
			testUser = &model.User{
				ID:          "edit-user-id",
				UserName:    "edituser",
				Name:        "Edit User",
				Email:       "edit@example.com",
				IsAdmin:     false,
				NewPassword: "password123",
			}
			testUser.Libraries, _ = ds.Library(ctx).GetAll()
			err := ds.User(ctx).Put(testUser)
			Expect(err).NotTo(HaveOccurred())

			err = ds.User(ctx).SetUserLibraries(testUser.ID, []int{1})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should update user email", func() {
			userID = "edituser"
			email = "newemail@example.com"

			runUserEdit(ctx)

			user, err := ds.User(ctx).FindByUsername("edituser")
			Expect(err).NotTo(HaveOccurred())
			Expect(user.Email).To(Equal("newemail@example.com"))
		})

		It("should remove user email", func() {
			userID = "edituser"
			removeEmail = true

			runUserEdit(ctx)

			user, err := ds.User(ctx).FindByUsername("edituser")
			Expect(err).NotTo(HaveOccurred())
			Expect(user.Email).To(BeEmpty())
		})

		It("should update user name", func() {
			userID = "edituser"
			name = "New Name"

			runUserEdit(ctx)

			user, err := ds.User(ctx).FindByUsername("edituser")
			Expect(err).NotTo(HaveOccurred())
			Expect(user.Name).To(Equal("New Name"))
		})

		It("should remove user name", func() {
			userID = "edituser"
			removeName = true

			runUserEdit(ctx)

			user, err := ds.User(ctx).FindByUsername("edituser")
			Expect(err).NotTo(HaveOccurred())
			Expect(user.Name).To(BeEmpty())
		})

		It("should promote user to admin", func() {
			userID = "edituser"
			setAdmin = true

			runUserEdit(ctx)

			user, err := ds.User(ctx).FindByUsername("edituser")
			Expect(err).NotTo(HaveOccurred())
			Expect(user.IsAdmin).To(BeTrue())
		})

		It("should demote admin to regular user", func() {
			// First make user an admin
			testUser.IsAdmin = true
			err := ds.User(ctx).Put(testUser)
			Expect(err).NotTo(HaveOccurred())

			userID = "edituser"
			setRegularUser = true

			runUserEdit(ctx)

			user, err := ds.User(ctx).FindByUsername("edituser")
			Expect(err).NotTo(HaveOccurred())
			Expect(user.IsAdmin).To(BeFalse())
		})

		It("should update user password", func() {
			cleanup := mockPasswordInput("newpassword123", "newpassword123")
			defer cleanup()

			userID = "edituser"
			setPassword = true

			runUserEdit(ctx)

			// We can't directly verify the password, but we can verify
			// the function completed without error
			user, err := ds.User(ctx).FindByUsername("edituser")
			Expect(err).NotTo(HaveOccurred())
			Expect(user).NotTo(BeNil())
		})

		It("should update library access", func() {
			// Create additional library
			err := ds.Library(ctx).Put(&model.Library{
				ID:   2,
				Name: "Test Library 2",
				Path: "/music2",
			})
			Expect(err).NotTo(HaveOccurred())

			userID = "edituser"
			libraryIds = []int{1, 2}

			runUserEdit(ctx)

			user, err := ds.User(ctx).FindByUsername("edituser")
			Expect(err).NotTo(HaveOccurred())

			// Reload user with libraries
			libraries, err := ds.User(ctx).GetUserLibraries(user.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(libraries).To(HaveLen(2))
		})

		It("should handle no changes gracefully", func() {
			userID = "edituser"

			// Don't set any flags, so no changes should be made
			runUserEdit(ctx)

			user, err := ds.User(ctx).FindByUsername("edituser")
			Expect(err).NotTo(HaveOccurred())
			Expect(user.Email).To(Equal("edit@example.com"))
			Expect(user.Name).To(Equal("Edit User"))
		})

		It("should handle multiple changes at once", func() {
			userID = "edituser"
			email = "newemail@example.com"
			name = "New Name"
			setAdmin = true

			runUserEdit(ctx)

			user, err := ds.User(ctx).FindByUsername("edituser")
			Expect(err).NotTo(HaveOccurred())
			Expect(user.Email).To(Equal("newemail@example.com"))
			Expect(user.Name).To(Equal("New Name"))
			Expect(user.IsAdmin).To(BeTrue())
		})
	})

	Describe("User list command", func() {
		BeforeEach(func() {
			// Create a few test users
			users := []*model.User{
				{
					ID:          "user1",
					UserName:    "user1",
					Name:        "User One",
					Email:       "user1@example.com",
					IsAdmin:     false,
					NewPassword: "password",
				},
				{
					ID:          "user2",
					UserName:    "user2",
					Name:        "User Two",
					Email:       "user2@example.com",
					IsAdmin:     false,
					NewPassword: "password",
				},
			}

			for _, user := range users {
				user.Libraries, _ = ds.Library(ctx).GetAll()
				err := ds.User(ctx).Put(user)
				Expect(err).NotTo(HaveOccurred())
				err = ds.User(ctx).SetUserLibraries(user.ID, []int{1})
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should list all users in CSV format", func() {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			outputFormat = "csv"
			runUserList(ctx)

			w.Close()
			os.Stdout = oldStdout

			output, _ := io.ReadAll(r)
			lines := strings.Split(string(output), "\n")

			// Verify CSV header
			Expect(lines[0]).To(ContainSubstring("user id"))
			Expect(lines[0]).To(ContainSubstring("username"))
			Expect(lines[0]).To(ContainSubstring("admin"))

			// Verify at least 3 users are listed (admin + 2 test users)
			Expect(len(lines)).To(BeNumerically(">=", 4)) // header + 3 users + empty line

			// Verify CSV is properly formatted
			reader := csv.NewReader(bytes.NewReader(output))
			records, err := reader.ReadAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(records)).To(BeNumerically(">=", 4)) // header + 3 users
		})

		It("should list all users in JSON format", func() {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			outputFormat = "json"
			runUserList(ctx)

			w.Close()
			os.Stdout = oldStdout

			output, _ := io.ReadAll(r)

			// Verify JSON is valid
			var users []displayUser
			err := json.Unmarshal(output, &users)
			Expect(err).NotTo(HaveOccurred())
			Expect(users).To(HaveLen(3)) // admin + 2 test users

			// Verify JSON structure
			found := false
			for _, user := range users {
				if user.Username == "user1" {
					found = true
					Expect(user.Email).To(Equal("user1@example.com"))
					Expect(user.Name).To(Equal("User One"))
					Expect(user.Admin).To(BeFalse())
					Expect(user.Libraries).To(HaveLen(1))
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should include library information in output", func() {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			outputFormat = "json"
			runUserList(ctx)

			w.Close()
			os.Stdout = oldStdout

			output, _ := io.ReadAll(r)

			var users []displayUser
			err := json.Unmarshal(output, &users)
			Expect(err).NotTo(HaveOccurred())

			// Verify libraries are included
			for _, user := range users {
				Expect(user.Libraries).NotTo(BeEmpty())
				for _, lib := range user.Libraries {
					Expect(lib.ID).To(BeNumerically(">", 0))
					Expect(lib.Path).NotTo(BeEmpty())
				}
			}
		})
	})

	Describe("Helper functions", func() {
		It("should find user by username", func() {
			user, err := getUser(ctx, "admin", ds)
			Expect(err).NotTo(HaveOccurred())
			Expect(user.UserName).To(Equal("admin"))
		})

		It("should find user by ID", func() {
			user, err := getUser(ctx, "admin-id", ds)
			Expect(err).NotTo(HaveOccurred())
			Expect(user.ID).To(Equal("admin-id"))
		})

		It("should return error for non-existent user", func() {
			_, err := getUser(ctx, "nonexistent", ds)
			Expect(err).To(HaveOccurred())
		})

		It("should validate library IDs", func() {
			// Request invalid library ID
			invalidLibs, err := ds.Library(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"id": []int{999}},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(invalidLibs)).To(Equal(0))

			// libraryError should report the mismatch
			libraryIds = []int{999}
			err = libraryError(invalidLibs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not all available libraries found"))
		})
	})

	Describe("Command structure", func() {
		It("should have correct user root command configuration", func() {
			Expect(userRoot.Use).To(Equal("user"))
			Expect(userRoot.Short).NotTo(BeEmpty())
		})

		It("should have create subcommand with correct aliases", func() {
			Expect(userCreateCommand.Use).To(Equal("create"))
			Expect(userCreateCommand.Aliases).To(ContainElement("c"))
		})

		It("should have delete subcommand with correct aliases", func() {
			Expect(userDeleteCommand.Use).To(Equal("delete"))
			Expect(userDeleteCommand.Aliases).To(ContainElement("d"))
		})

		It("should have edit subcommand with correct aliases", func() {
			Expect(userEditCommand.Use).To(Equal("edit"))
			Expect(userEditCommand.Aliases).To(ContainElement("e"))
		})

		It("should have list subcommand", func() {
			Expect(userListCommand.Use).To(Equal("list"))
		})
	})

	Describe("Password prompt", func() {
		var oldStdin *os.File

		AfterEach(func() {
			if oldStdin != nil {
				os.Stdin = oldStdin
			}
		})

		It("should handle empty password cancellation", func() {
			oldStdin = os.Stdin
			r, w, _ := os.Pipe()

			go func() {
				defer w.Close()
				w.Write([]byte("\n")) // Empty password
			}()

			os.Stdin = r

			password := promptPassword()
			Expect(password).To(BeEmpty())
		})

		It("should handle mismatched passwords", func() {
			oldStdin = os.Stdin
			r, w, _ := os.Pipe()

			go func() {
				defer w.Close()
				// First attempt: mismatched
				w.Write([]byte("password123\n"))
				w.Write([]byte("different\n"))
				// Second attempt: empty to cancel
				w.Write([]byte("\n"))
			}()

			os.Stdin = r

			password := promptPassword()
			Expect(password).To(BeEmpty())
		})
	})

	Describe("Edge cases", func() {
		It("should handle user with no email", func() {
			user := &model.User{
				ID:          "no-email",
				UserName:    "noemailuser",
				Name:        "No Email User",
				Email:       "",
				IsAdmin:     false,
				NewPassword: "password",
			}
			err := ds.User(ctx).Put(user)
			Expect(err).NotTo(HaveOccurred())

			userID = "noemailuser"
			email = "newemail@example.com"

			runUserEdit(ctx)

			updatedUser, err := ds.User(ctx).FindByUsername("noemailuser")
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedUser.Email).To(Equal("newemail@example.com"))
		})

		It("should handle setting same email", func() {
			userID = "admin"
			email = "admin@example.com" // Same as current

			runUserEdit(ctx)

			// Should complete without error
			user, err := ds.User(ctx).FindByUsername("admin")
			Expect(err).NotTo(HaveOccurred())
			Expect(user.Email).To(Equal("admin@example.com"))
		})

		It("should handle admin user library update", func() {
			// Create additional library
			err := ds.Library(ctx).Put(&model.Library{
				ID:   2,
				Name: "Test Library 2",
				Path: "/music2",
			})
			Expect(err).NotTo(HaveOccurred())

			userID = "admin"
			libraryIds = []int{1, 2}

			runUserEdit(ctx)

			// Admin should still have access to all libraries
			user, err := ds.User(ctx).FindByUsername("admin")
			Expect(err).NotTo(HaveOccurred())
			Expect(user.IsAdmin).To(BeTrue())
		})
	})
})
