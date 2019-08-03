package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	. "github.com/AlekSi/pointer"

	"github.com/monitoror/monitoror/monitorable/travisci"

	mErrors "github.com/monitoror/monitoror/models/errors"
	. "github.com/monitoror/monitoror/models/tiles"
	"github.com/monitoror/monitoror/monitorable/travisci/mocks"
	"github.com/monitoror/monitoror/monitorable/travisci/models"
	"github.com/stretchr/testify/assert"
	. "github.com/stretchr/testify/mock"
)

var group, repo, branch = "test", "test", "master"

func TestBuild_Error_NoHost(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetLastBuildStatus", AnythingOfType("string"), AnythingOfType("string"), AnythingOfType("string")).
		Return(nil, errors.New("no such host"))

	tu := NewTravisCIUsecase(mockRepository)

	tile, err := tu.Build(&models.BuildParams{Group: group, Repository: repo, Branch: branch})
	if assert.Error(t, err) {
		assert.Nil(t, tile)
		assert.IsType(t, &mErrors.TimeoutError{}, err)
		mockRepository.AssertNumberOfCalls(t, "GetLastBuildStatus", 1)
		mockRepository.AssertExpectations(t)
	}
}

func TestBuild_Error_NoNetwork(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetLastBuildStatus", AnythingOfType("string"), AnythingOfType("string"), AnythingOfType("string")).
		Return(nil, errors.New("dial tcp: lookup"))

	tu := NewTravisCIUsecase(mockRepository)

	tile, err := tu.Build(&models.BuildParams{Group: group, Repository: repo, Branch: branch})
	if assert.Error(t, err) {
		assert.Nil(t, tile)
		assert.IsType(t, &mErrors.TimeoutError{}, err)
		mockRepository.AssertNumberOfCalls(t, "GetLastBuildStatus", 1)
		mockRepository.AssertExpectations(t)
	}
}

func TestBuild_Timeout(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetLastBuildStatus", AnythingOfType("string"), AnythingOfType("string"), AnythingOfType("string")).
		Return(nil, context.DeadlineExceeded)

	tu := NewTravisCIUsecase(mockRepository)

	tile, err := tu.Build(&models.BuildParams{Group: group, Repository: repo, Branch: branch})
	if assert.Error(t, err) {
		assert.Nil(t, tile)
		assert.IsType(t, &mErrors.TimeoutError{}, err)
		mockRepository.AssertNumberOfCalls(t, "GetLastBuildStatus", 1)
		mockRepository.AssertExpectations(t)
	}
}

func TestBuild_Error_System(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetLastBuildStatus", AnythingOfType("string"), AnythingOfType("string"), AnythingOfType("string")).
		Return(nil, errors.New("boom"))

	tu := NewTravisCIUsecase(mockRepository)

	tile, err := tu.Build(&models.BuildParams{Group: group, Repository: repo, Branch: branch})
	if assert.Error(t, err) {
		assert.Nil(t, tile)
		assert.IsType(t, &mErrors.SystemError{}, err)
		mockRepository.AssertNumberOfCalls(t, "GetLastBuildStatus", 1)
		mockRepository.AssertExpectations(t)
	}
}

func TestBuild_Error_NoBuild(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetLastBuildStatus", AnythingOfType("string"), AnythingOfType("string"), AnythingOfType("string")).
		Return(nil, nil)

	tu := NewTravisCIUsecase(mockRepository)

	tile, err := tu.Build(&models.BuildParams{Group: group, Repository: repo, Branch: branch})
	if assert.Error(t, err) {
		assert.Nil(t, tile)
		assert.IsType(t, &mErrors.NoBuildError{}, err)
		mockRepository.AssertNumberOfCalls(t, "GetLastBuildStatus", 1)
		mockRepository.AssertExpectations(t)
	}
}

func TestBuild_Success(t *testing.T) {
	build := buildResponse(branch, "passed", time.Now(), time.Now(), time.Second*100)

	mockRepository := new(mocks.Repository)
	mockRepository.On("GetLastBuildStatus", AnythingOfType("string"), AnythingOfType("string"), AnythingOfType("string")).
		Return(build, nil)

	tu := NewTravisCIUsecase(mockRepository)
	tUsecase, ok := tu.(*travisCIUsecase)
	if assert.True(t, ok, "enable to case tu into travisCIUsecase") {
		// Expected
		expected := NewBuildTile(travisci.TravisCIBuildTileType)
		expected.Label = fmt.Sprintf("%s : #%s", repo, branch)
		expected.Status = parseState(build.State)
		expected.PreviousStatus = SuccessStatus
		expected.StartedAt = ToTime(build.StartedAt)
		expected.FinishedAt = ToTime(build.FinishedAt)
		expected.Author = &Author{
			Name:      build.Author.Name,
			AvatarUrl: build.Author.AvatarUrl,
		}

		// Tests
		tUsecase.buildsCache.Add(fmt.Sprintf("%s : #%s", repo, branch), "0", SuccessStatus, time.Second*120)
		tile, err := tu.Build(&models.BuildParams{Group: group, Repository: repo, Branch: branch})
		if assert.NotNil(t, tile) {
			assert.NoError(t, err)
			assert.Equal(t, expected, tile)

			// Check if duration is added into cache
			previousDuration := tUsecase.buildsCache.GetEstimatedDuration(tile.Label)
			assert.NotNil(t, previousDuration)
			assert.Equal(t, time.Second*110, *previousDuration)

			mockRepository.AssertNumberOfCalls(t, "GetLastBuildStatus", 1)
			mockRepository.AssertExpectations(t)
		}
	}
}

func TestBuild_Failed(t *testing.T) {
	build := buildResponse(branch, "failed", time.Now(), time.Now(), time.Second*100)

	mockRepository := new(mocks.Repository)
	mockRepository.On("GetLastBuildStatus", AnythingOfType("string"), AnythingOfType("string"), AnythingOfType("string")).
		Return(build, nil)

	tu := NewTravisCIUsecase(mockRepository)
	tUsecase, ok := tu.(*travisCIUsecase)
	if assert.True(t, ok, "enable to case tu into travisCIUsecase") {
		// Expected
		expected := NewBuildTile(travisci.TravisCIBuildTileType)
		expected.Label = fmt.Sprintf("%s : #%s", repo, branch)
		expected.Status = parseState(build.State)
		expected.PreviousStatus = SuccessStatus
		expected.StartedAt = ToTime(build.StartedAt)
		expected.FinishedAt = ToTime(build.FinishedAt)
		expected.Author = &Author{
			Name:      build.Author.Name,
			AvatarUrl: build.Author.AvatarUrl,
		}

		tUsecase.buildsCache.Add(fmt.Sprintf("%s : #%s", repo, branch), "0", SuccessStatus, time.Second*120)
		tile, err := tu.Build(&models.BuildParams{Group: group, Repository: repo, Branch: branch})
		if assert.NotNil(t, tile) {
			assert.NoError(t, err)
			assert.Equal(t, expected, tile)

			// Check if duration is added into cache
			previousDuration := tUsecase.buildsCache.GetEstimatedDuration(tile.Label)
			assert.NotNil(t, previousDuration)
			assert.Equal(t, time.Second*110, *previousDuration)

			mockRepository.AssertNumberOfCalls(t, "GetLastBuildStatus", 1)
			mockRepository.AssertExpectations(t)
		}
	}
}

func TestBuild_Queued(t *testing.T) {
	build := buildResponse(branch, "received", time.Now(), time.Time{}, 100)

	mockRepository := new(mocks.Repository)
	mockRepository.On("GetLastBuildStatus", AnythingOfType("string"), AnythingOfType("string"), AnythingOfType("string")).
		Return(build, nil)

	tu := NewTravisCIUsecase(mockRepository)
	tUsecase, ok := tu.(*travisCIUsecase)
	if assert.True(t, ok) {
		tUsecase.buildsCache.Add(fmt.Sprintf("%s : #%s", repo, branch), "0", SuccessStatus, time.Second*10)

		// Expected
		expected := NewBuildTile(travisci.TravisCIBuildTileType)
		expected.Label = fmt.Sprintf("%s : #%s", repo, branch)
		expected.Status = parseState(build.State)
		expected.PreviousStatus = SuccessStatus
		expected.StartedAt = ToTime(build.StartedAt)
		expected.Author = &Author{
			Name:      build.Author.Name,
			AvatarUrl: build.Author.AvatarUrl,
		}

		// Without Estimated Duration
		tile, err := tu.Build(&models.BuildParams{Group: group, Repository: repo, Branch: branch})
		if assert.NotNil(t, tile) {
			assert.NoError(t, err)
			assert.Equal(t, expected, tile)
		}
	}
}

func TestBuild_Running(t *testing.T) {
	build := buildResponse(branch, "started", time.Now(), time.Time{}, 100)

	mockRepository := new(mocks.Repository)
	mockRepository.On("GetLastBuildStatus", AnythingOfType("string"), AnythingOfType("string"), AnythingOfType("string")).
		Return(build, nil)

	tu := NewTravisCIUsecase(mockRepository)
	tUsecase, ok := tu.(*travisCIUsecase)
	if assert.True(t, ok, "enable to case tu into travisCIUsecase") {
		// Expected
		expected := NewBuildTile(travisci.TravisCIBuildTileType)
		expected.Label = fmt.Sprintf("%s : #%s", repo, branch)
		expected.Status = parseState(build.State)
		expected.PreviousStatus = UnknownStatus
		expected.Duration = ToInt64(int64(build.Duration / time.Second))
		expected.EstimatedDuration = ToInt64(int64(0))
		expected.StartedAt = ToTime(build.StartedAt)
		expected.Author = &Author{
			Name:      build.Author.Name,
			AvatarUrl: build.Author.AvatarUrl,
		}

		// Without Previous Build
		tile, err := tu.Build(&models.BuildParams{Group: group, Repository: repo, Branch: branch})
		if assert.NotNil(t, tile) {
			assert.NoError(t, err)
			assert.Equal(t, expected, tile)
		}

		// With Previous Build
		expected.PreviousStatus = SuccessStatus
		expected.EstimatedDuration = ToInt64(int64(120))
		tUsecase.buildsCache.Add(fmt.Sprintf("%s : #%s", repo, branch), "0", SuccessStatus, time.Second*120)

		tile, err = tu.Build(&models.BuildParams{Group: group, Repository: repo, Branch: branch})
		if assert.NotNil(t, tile) {
			assert.NoError(t, err)
			assert.Equal(t, expected, tile)
		}

		mockRepository.AssertNumberOfCalls(t, "GetLastBuildStatus", 2)
		mockRepository.AssertExpectations(t)
	}
}

func TestBuild_Aborded(t *testing.T) {
	build := buildResponse(branch, "canceled", time.Now(), time.Time{}, 100)

	mockRepository := new(mocks.Repository)
	mockRepository.On("GetLastBuildStatus", AnythingOfType("string"), AnythingOfType("string"), AnythingOfType("string")).
		Return(build, nil)

	tu := NewTravisCIUsecase(mockRepository)
	tUsecase, ok := tu.(*travisCIUsecase)
	if assert.True(t, ok) {
		tUsecase.buildsCache.Add(fmt.Sprintf("%s : #%s", repo, branch), "0", SuccessStatus, time.Second*10)

		// Expected
		expected := NewBuildTile(travisci.TravisCIBuildTileType)
		expected.Label = fmt.Sprintf("%s : #%s", repo, branch)
		expected.Status = parseState(build.State)
		expected.PreviousStatus = SuccessStatus
		expected.StartedAt = ToTime(build.StartedAt)
		expected.Author = &Author{
			Name:      build.Author.Name,
			AvatarUrl: build.Author.AvatarUrl,
		}

		// Without Estimated Duration
		tile, err := tu.Build(&models.BuildParams{Group: group, Repository: repo, Branch: branch})
		if assert.NotNil(t, tile) {
			assert.NoError(t, err)
			assert.Equal(t, expected, tile)
		}
	}
}

func TestParseState(t *testing.T) {
	assert.Equal(t, QueuedStatus, parseState("created"))
	assert.Equal(t, QueuedStatus, parseState("received"))
	assert.Equal(t, RunningStatus, parseState("started"))
	assert.Equal(t, SuccessStatus, parseState("passed"))
	assert.Equal(t, FailedStatus, parseState("failed"))
	assert.Equal(t, FailedStatus, parseState("errored"))
	assert.Equal(t, AbortedStatus, parseState("canceled"))
	assert.Equal(t, UnknownStatus, parseState(""))
}

func buildResponse(branch, state string, startedAt, finishedAt time.Time, duration time.Duration) *models.Build {
	return &models.Build{
		Id:     1,
		Branch: branch,
		Author: models.Author{
			Name:      "me",
			AvatarUrl: "http://avatar.com",
		},
		State:      state,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Duration:   duration,
	}
}