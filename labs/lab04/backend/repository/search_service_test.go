package repository

import (
	"context"
	"testing"

	"lab04-backend/database"

	squirrel "github.com/Masterminds/squirrel"
)

// TestSearchService tests the Squirrel query builder approach
func TestSearchService(t *testing.T) {
	db, err := database.InitDB()
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDB(db)

	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	searchService := NewSearchService(db)

	t.Run("SearchPosts with filters", func(t *testing.T) {
		// Пример простого теста поиска постов с фильтрами
		// Проверим хотя бы пустой фильтр и фильтр по строке запроса

		// Очистим и подготовим данные
		_, err := db.Exec("DELETE FROM posts")
		if err != nil {
			t.Fatalf("Failed to clean posts table: %v", err)
		}

		_, err = db.Exec(`
			INSERT INTO posts (user_id, title, content, published, created_at, updated_at)
			VALUES (1, 'Golang post', 'Content 1', true, NOW(), NOW()),
				   (2, 'Python post', 'Content 2', false, NOW(), NOW()),
				   (1, 'Another Golang post', 'Content 3', true, NOW(), NOW())
		`)
		if err != nil {
			t.Fatalf("Failed to insert test posts: %v", err)
		}

		// Пустой фильтр - должны получить все
		posts, err := searchService.SearchPosts(context.Background(), SearchFilters{})
		if err != nil {
			t.Fatalf("SearchPosts failed: %v", err)
		}
		if len(posts) != 3 {
			t.Errorf("expected 3 posts, got %d", len(posts))
		}

		// Поиск по строке
		posts, err = searchService.SearchPosts(context.Background(), SearchFilters{Query: "Golang"})
		if err != nil {
			t.Fatalf("SearchPosts failed: %v", err)
		}
		if len(posts) != 2 {
			t.Errorf("expected 2 posts with 'Golang', got %d", len(posts))
		}
	})

	t.Run("SearchUsers", func(t *testing.T) {
		// Простейший тест для поиска пользователей
		// Очистка и подготовка тестовых данных
		_, err := db.Exec("DELETE FROM users")
		if err != nil {
			t.Fatalf("Failed to clean users table: %v", err)
		}

		_, err = db.Exec(`
			INSERT INTO users (name, email, created_at, updated_at)
			VALUES ('Alice', 'alice@example.com', NOW(), NOW()),
				   ('Bob', 'bob@example.com', NOW(), NOW()),
				   ('Alicia', 'alicia@example.com', NOW(), NOW())
		`)
		if err != nil {
			t.Fatalf("Failed to insert test users: %v", err)
		}

		users, err := searchService.SearchUsers(context.Background(), "Ali", 10)
		if err != nil {
			t.Fatalf("SearchUsers failed: %v", err)
		}
		if len(users) != 2 {
			t.Errorf("expected 2 users matching 'Ali', got %d", len(users))
		}
	})

	t.Run("GetPostStats", func(t *testing.T) {
		// Для агрегации вставим тестовые данные и вызовем метод
		err := database.RunMigrations(db)
		if err != nil {
			t.Fatalf("Failed to run migrations: %v", err)
		}

		stats, err := searchService.GetPostStats(context.Background())
		if err != nil {
			t.Fatalf("GetPostStats failed: %v", err)
		}
		// Просто проверим что получили хоть какие-то данные (можно расширять)
		if stats == nil {
			t.Error("expected non-nil stats result")
		}
	})

	t.Run("GetTopUsers", func(t *testing.T) {
		topUsers, err := searchService.GetTopUsers(context.Background(), 5)
		if err != nil {
			t.Fatalf("GetTopUsers failed: %v", err)
		}
		if len(topUsers) > 5 {
			t.Errorf("expected at most 5 top users, got %d", len(topUsers))
		}
	})

	t.Run("BuildDynamicQuery", func(t *testing.T) {
		baseQuery := searchService.psql.Select("*").From("posts")
		filters := SearchFilters{Query: "test", Published: &[]bool{true}[0]}
		query := searchService.BuildDynamicQuery(baseQuery, filters)
		sql, args, err := query.ToSql()
		if err != nil {
			t.Fatalf("BuildDynamicQuery ToSql failed: %v", err)
		}
		if sql == "" {
			t.Error("generated SQL is empty")
		}
		if len(args) == 0 {
			t.Error("expected arguments in query")
		}
	})
}

// TestSquirrelQueryBuilder tests Squirrel query building functionality
func TestSquirrelQueryBuilder(t *testing.T) {
	t.Run("Basic Query Building", func(t *testing.T) {
		psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
		query := psql.Select("id", "name").From("users").Where(squirrel.Eq{"active": true})
		sql, args, err := query.ToSql()
		if err != nil {
			t.Fatalf("ToSql failed: %v", err)
		}
		expectedSQL := "SELECT id, name FROM users WHERE active = $1"
		if sql != expectedSQL {
			t.Errorf("expected SQL %q, got %q", expectedSQL, sql)
		}
		if len(args) != 1 || args[0] != true {
			t.Errorf("expected args [true], got %v", args)
		}
	})

	t.Run("Complex Query Building", func(t *testing.T) {
		psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
		subquery := psql.Select("user_id").From("posts").Where(squirrel.Eq{"published": true})
		query := psql.Select("u.id", "u.name").
			From("users u").
			JoinClause("JOIN (?) p ON p.user_id = u.id", subquery).
			Where(squirrel.Or{
				squirrel.Like{"u.name": "%admin%"},
				squirrel.Eq{"u.active": true},
			}).
			OrderBy("u.name").
			Limit(10)
		sql, args, err := query.ToSql()
		if err != nil {
			t.Fatalf("Complex query ToSql failed: %v", err)
		}
		if sql == "" || len(args) == 0 {
			t.Error("expected non-empty SQL and args")
		}
	})
}

// BenchmarkSquirrelVsManualSQL benchmarks Squirrel vs manual SQL building
func BenchmarkSquirrelVsManualSQL(b *testing.B) {
	b.Run("Squirrel", func(b *testing.B) {
		psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
		for i := 0; i < b.N; i++ {
			_ = psql.Select("id", "name").
				From("users").
				Where(squirrel.Eq{"active": true}).
				OrderBy("name ASC").
				Limit(10)
		}
	})

	b.Run("Manual SQL", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = "SELECT id, name FROM users WHERE active = $1 ORDER BY name ASC LIMIT 10"
		}
	})
}
