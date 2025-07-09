package repository

import (
	"testing"
	"time"

	"lab04-backend/models"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB создаёт in-memory БД и миграции для тестов
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(&models.Category{}, &models.Post{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

// Пример реализации CategoryRepository (если нужно, поправь под свой код)
type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(category *models.Category) error {
	return r.db.Create(category).Error
}

func (r *CategoryRepository) GetByID(id uint) (*models.Category, error) {
	var cat models.Category
	err := r.db.First(&cat, id).Error
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

func (r *CategoryRepository) GetAll() ([]models.Category, error) {
	var cats []models.Category
	err := r.db.Order("name").Find(&cats).Error
	return cats, err
}

func (r *CategoryRepository) Update(category *models.Category) error {
	return r.db.Save(category).Error
}

func (r *CategoryRepository) Delete(id uint) error {
	return r.db.Delete(&models.Category{}, id).Error
}

func (r *CategoryRepository) FindByName(name string) (*models.Category, error) {
	var cat models.Category
	err := r.db.Where("name = ?", name).First(&cat).Error
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

func TestCategoryRepository(t *testing.T) {
	db := setupTestDB(t)
	categoryRepo := NewCategoryRepository(db)

	t.Run("Create category with GORM", func(t *testing.T) {
		cat := &models.Category{
			Name:        "Technology",
			Description: "Tech-related posts",
			Color:       "#007bff",
		}
		err := categoryRepo.Create(cat)
		assert.NoError(t, err)
		assert.NotZero(t, cat.ID)
		assert.WithinDuration(t, time.Now(), cat.CreatedAt, time.Second*5)
	})

	t.Run("GetByID with GORM", func(t *testing.T) {
		cat := &models.Category{Name: "TestCat"}
		err := categoryRepo.Create(cat)
		assert.NoError(t, err)

		got, err := categoryRepo.GetByID(cat.ID)
		assert.NoError(t, err)
		assert.Equal(t, cat.Name, got.Name)

		_, err = categoryRepo.GetByID(9999)
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("GetAll with GORM", func(t *testing.T) {
		// Создадим несколько категорий
		_ = categoryRepo.Create(&models.Category{Name: "Category B"})
		_ = categoryRepo.Create(&models.Category{Name: "Category A"})
		_ = categoryRepo.Create(&models.Category{Name: "Category C"})

		cats, err := categoryRepo.GetAll()
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(cats), 3)
		assert.Equal(t, "Category A", cats[0].Name) // Проверяем сортировку по имени
	})

	t.Run("Update with GORM", func(t *testing.T) {
		cat := &models.Category{Name: "OldName"}
		_ = categoryRepo.Create(cat)

		originalUpdatedAt := cat.UpdatedAt

		cat.Name = "UpdatedName"
		err := categoryRepo.Update(cat)
		assert.NoError(t, err)
		assert.Equal(t, "UpdatedName", cat.Name)
		assert.True(t, cat.UpdatedAt.After(originalUpdatedAt) || cat.UpdatedAt.Equal(originalUpdatedAt))
	})

	t.Run("Delete with GORM", func(t *testing.T) {
		cat := &models.Category{Name: "ToDelete"}
		_ = categoryRepo.Create(cat)

		err := categoryRepo.Delete(cat.ID)
		assert.NoError(t, err)

		_, err = categoryRepo.GetByID(cat.ID)
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("FindByName with GORM", func(t *testing.T) {
		cat := &models.Category{Name: "UniqueName"}
		_ = categoryRepo.Create(cat)

		got, err := categoryRepo.FindByName("UniqueName")
		assert.NoError(t, err)
		assert.Equal(t, "UniqueName", got.Name)

		_, err = categoryRepo.FindByName("NonExistingName")
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})
}
