package categoryapp

import (
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	categorycontract "github.com/dujiao-next/internal/modules/catalog/category/contract"
	categorydomain "github.com/dujiao-next/internal/modules/catalog/category/domain"
	"github.com/dujiao-next/internal/shared/jsonmap"
)

const categoryDescriptionMaxRunes = 500

var (
	ErrNotFound           = errors.New("category not found")
	ErrSlugExists         = errors.New("category slug exists")
	ErrParentInvalid      = errors.New("category parent invalid")
	ErrInUse              = errors.New("category in use")
	ErrDescriptionTooLong = errors.New("category description too long")
)

// Service 承载分类查询、创建、更新、启停和删除用例。
type Service struct {
	repository categorycontract.Repository
}

func NewService(repository categorycontract.Repository) *Service {
	return &Service{repository: repository}
}

// UpsertInput 是创建与更新分类的应用层输入。
type UpsertInput struct {
	ParentID        uint
	Slug            string
	NameJSON        map[string]interface{}
	Icon            string
	DescriptionJSON map[string]interface{}
	SeoMetaJSON     map[string]interface{}
	SortOrder       int
}

func (service *Service) List() ([]categorydomain.Category, error) {
	return service.repository.List()
}

func (service *Service) ListActive() ([]categorydomain.Category, error) {
	return service.repository.ListActive()
}

func (service *Service) SetActive(id string, active bool) (*categorydomain.Category, error) {
	category, err := service.repository.GetByID(id)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, ErrNotFound
	}
	if category.IsActive == active {
		return category, nil
	}
	if err := service.repository.UpdateActive(id, active); err != nil {
		return nil, err
	}
	category.IsActive = active
	return category, nil
}

func (service *Service) Create(input UpsertInput) (*categorydomain.Category, error) {
	description, err := normalizeDescription(input.DescriptionJSON)
	if err != nil {
		return nil, err
	}
	if err := service.validateParent(nil, input.ParentID); err != nil {
		return nil, err
	}
	count, err := service.repository.CountBySlug(input.Slug, nil)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrSlugExists
	}
	category := categorydomain.Category{
		ParentID:        input.ParentID,
		Slug:            input.Slug,
		NameJSON:        jsonmap.JSON(input.NameJSON),
		Icon:            input.Icon,
		DescriptionJSON: description,
		SeoMetaJSON:     jsonmap.JSON(input.SeoMetaJSON),
		SortOrder:       input.SortOrder,
		IsActive:        true,
	}
	if err := service.repository.Create(&category); err != nil {
		return nil, err
	}
	return &category, nil
}

func (service *Service) Update(id string, input UpsertInput) (*categorydomain.Category, error) {
	description, err := normalizeDescription(input.DescriptionJSON)
	if err != nil {
		return nil, err
	}
	category, err := service.repository.GetByID(id)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, ErrNotFound
	}
	if err := service.validateParent(category, input.ParentID); err != nil {
		return nil, err
	}
	count, err := service.repository.CountBySlug(input.Slug, &id)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrSlugExists
	}
	category.ParentID = input.ParentID
	category.Slug = input.Slug
	category.NameJSON = jsonmap.JSON(input.NameJSON)
	category.Icon = input.Icon
	// nil 表示调用方未提交该字段（旧版管理端），保留原值；
	// 空 map 表示显式清空。
	if input.DescriptionJSON != nil {
		category.DescriptionJSON = description
	}
	if input.SeoMetaJSON != nil {
		category.SeoMetaJSON = jsonmap.JSON(input.SeoMetaJSON)
	}
	category.SortOrder = input.SortOrder
	if err := service.repository.Update(category); err != nil {
		return nil, err
	}
	return category, nil
}

func (service *Service) Delete(id string) error {
	category, err := service.repository.GetByID(id)
	if err != nil {
		return err
	}
	if category == nil {
		return ErrNotFound
	}
	childCount, err := service.repository.CountChildren(id)
	if err != nil {
		return err
	}
	if childCount > 0 {
		return ErrInUse
	}
	count, err := service.repository.CountProducts(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrInUse
	}
	return service.repository.Delete(id)
}

// normalizeDescription 清洗多语言分类描述，并限制每种语言最多 500 个字符。
func normalizeDescription(raw map[string]interface{}) (jsonmap.JSON, error) {
	result := make(jsonmap.JSON)
	for lang, value := range raw {
		text, ok := value.(string)
		if !ok {
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if utf8.RuneCountInString(text) > categoryDescriptionMaxRunes {
			return nil, ErrDescriptionTooLong
		}
		result[lang] = text
	}
	return result, nil
}

func (service *Service) validateParent(category *categorydomain.Category, parentID uint) error {
	if parentID == 0 {
		return nil
	}
	if category != nil && category.ID == parentID {
		return ErrParentInvalid
	}
	parent, err := service.repository.GetByID(strconv.FormatUint(uint64(parentID), 10))
	if err != nil {
		return err
	}
	if parent == nil || parent.ParentID != 0 {
		return ErrParentInvalid
	}
	if category != nil && category.ParentID == 0 {
		childCount, err := service.repository.CountChildren(strconv.FormatUint(uint64(category.ID), 10))
		if err != nil {
			return err
		}
		if childCount > 0 {
			return ErrParentInvalid
		}
	}
	return nil
}
