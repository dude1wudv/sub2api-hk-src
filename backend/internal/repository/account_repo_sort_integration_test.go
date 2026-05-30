//go:build integration

package repository

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (s *AccountRepoSuite) TestList_DefaultSortByNameAsc() {
	mustCreateAccount(s.T(), s.client, &service.Account{Name: "z-account"})
	mustCreateAccount(s.T(), s.client, &service.Account{Name: "a-account"})

	accounts, _, err := s.repo.List(s.ctx, pagination.PaginationParams{Page: 1, PageSize: 10})
	s.Require().NoError(err)
	s.Require().Len(accounts, 2)
	s.Require().Equal("a-account", accounts[0].Name)
	s.Require().Equal("z-account", accounts[1].Name)
}

func (s *AccountRepoSuite) TestListWithFilters_SortByPriorityDesc() {
	mustCreateAccount(s.T(), s.client, &service.Account{Name: "low-priority", Priority: 10})
	mustCreateAccount(s.T(), s.client, &service.Account{Name: "high-priority", Priority: 90})

	accounts, _, err := s.repo.ListWithFilters(s.ctx, pagination.PaginationParams{
		Page:      1,
		PageSize:  10,
		SortBy:    "priority",
		SortOrder: "desc",
	}, "", "", "", "", 0, "")
	s.Require().NoError(err)
	s.Require().Len(accounts, 2)
	s.Require().Equal("high-priority", accounts[0].Name)
	s.Require().Equal("low-priority", accounts[1].Name)
}

func (s *AccountRepoSuite) TestListWithFilters_SortByQuotaRemainingDescUsesPlus5hAndFree7d() {
	plusGroup := mustCreateGroup(s.T(), s.client, &service.Group{Name: "plus+pro", Platform: service.PlatformOpenAI})
	freeGroup := mustCreateGroup(s.T(), s.client, &service.Group{Name: "free-special", Platform: service.PlatformOpenAI})
	plusHigh := mustCreateAccount(s.T(), s.client, &service.Account{
		Name:     "plus-high",
		Platform: service.PlatformOpenAI,
		Extra:    map[string]any{"codex_5h_used_percent": 10.0, "codex_7d_used_percent": 99.0},
	})
	freeMid := mustCreateAccount(s.T(), s.client, &service.Account{
		Name:     "free-mid",
		Platform: service.PlatformOpenAI,
		Extra:    map[string]any{"codex_5h_used_percent": 99.0, "codex_7d_used_percent": 40.0},
	})
	plusLow := mustCreateAccount(s.T(), s.client, &service.Account{
		Name:     "plus-low",
		Platform: service.PlatformOpenAI,
		Extra:    map[string]any{"codex_5h_used_percent": 80.0, "codex_7d_used_percent": 0.0},
	})
	mustBindAccountToGroup(s.T(), s.client, plusHigh.ID, plusGroup.ID, 50)
	mustBindAccountToGroup(s.T(), s.client, freeMid.ID, freeGroup.ID, 50)
	mustBindAccountToGroup(s.T(), s.client, plusLow.ID, plusGroup.ID, 50)

	accounts, _, err := s.repo.ListWithFilters(s.ctx, pagination.PaginationParams{
		Page:      1,
		PageSize:  10,
		SortBy:    "quota_remaining",
		SortOrder: "desc",
	}, service.PlatformOpenAI, "", "", "", 0, "")
	s.Require().NoError(err)
	s.Require().Len(accounts, 3)
	s.Require().Equal("plus-high", accounts[0].Name)
	s.Require().Equal("free-mid", accounts[1].Name)
	s.Require().Equal("plus-low", accounts[2].Name)
}
