package seeders

import (
	"business/tools/migrations/model"

	"gorm.io/gorm"
)

// CreateKeywordGroupWordLink はキーワードグループのサンプルデータを投入する。
func CreateKeywordGroupWordLink(tx *gorm.DB) error {
	var err error
	KeywordGroupWordLinks := []model.KeywordGroupWordLink{
		{
			KeywordGroupID: 1,
			KeyWordID:      1,
		},
		{
			KeywordGroupID: 1,
			KeyWordID:      2,
		},
		{
			KeywordGroupID: 2,
			KeyWordID:      4,
		},
		{
			KeywordGroupID: 3,
			KeyWordID:      7,
		},
		{
			KeywordGroupID: 3,
			KeyWordID:      9,
		},
		{
			KeywordGroupID: 4,
			KeyWordID:      11,
		},
		{
			KeywordGroupID: 4,
			KeyWordID:      13,
		},
		{
			KeywordGroupID: 5,
			KeyWordID:      15,
		},
		{
			KeywordGroupID: 5,
			KeyWordID:      16,
		},
		{
			KeywordGroupID: 6,
			KeyWordID:      18,
		},
		{
			KeywordGroupID: 6,
			KeyWordID:      20,
		},
		{
			KeywordGroupID: 7,
			KeyWordID:      21,
		},
		{
			KeywordGroupID: 7,
			KeyWordID:      22,
		},
		{
			KeywordGroupID: 7,
			KeyWordID:      23,
		},
		{
			KeywordGroupID: 8,
			KeyWordID:      24,
		},
		{
			KeywordGroupID: 8,
			KeyWordID:      25,
		},
		{
			KeywordGroupID: 8,
			KeyWordID:      26,
		},
		{
			KeywordGroupID: 9,
			KeyWordID:      27,
		},
		{
			KeywordGroupID: 10,
			KeyWordID:      29,
		},
		{
			KeywordGroupID: 10,
			KeyWordID:      30,
		},
		{
			KeywordGroupID: 10,
			KeyWordID:      31,
		},
		{
			KeywordGroupID: 11,
			KeyWordID:      32,
		},
		{
			KeywordGroupID: 11,
			KeyWordID:      33,
		},
		{
			KeywordGroupID: 12,
			KeyWordID:      34,
		},
		{
			KeywordGroupID: 12,
			KeyWordID:      35,
		},
		{
			KeywordGroupID: 13,
			KeyWordID:      37,
		},
		{
			KeywordGroupID: 14,
			KeyWordID:      39,
		},

		{
			KeywordGroupID: 15,
			KeyWordID:      42,
		},
		{
			KeywordGroupID: 15,
			KeyWordID:      43,
		},
		{
			KeywordGroupID: 15,
			KeyWordID:      44,
		},
		{
			KeywordGroupID: 16,
			KeyWordID:      45,
		},
		{
			KeywordGroupID: 16,
			KeyWordID:      46,
		},
		{
			KeywordGroupID: 16,
			KeyWordID:      47,
		},
		{
			KeywordGroupID: 17,
			KeyWordID:      48,
		},
		{
			KeywordGroupID: 17,
			KeyWordID:      49,
		},
		{
			KeywordGroupID: 17,
			KeyWordID:      50,
		},
	}

	for _, KeywordGroupWordLink := range KeywordGroupWordLinks {
		err := tx.Create(&KeywordGroupWordLink).Error
		if err != nil {
			return err
		}
	}

	return err
}
