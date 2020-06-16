package runner

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejohn/go-atomic/art"
)

func newRunner(location string) (*Runner, error) {
	ar := &Runner{
		AtomicsFolder: location,
	}
	err := ar.LoadTechniques()
	return ar, err
}

func TestNewRunner(t *testing.T) {
	ar, err := newRunner(testFolder)
	require.NoError(t, err)
	tests := ar.GetAllTechniques()
	assert.Equal(t, 4, len(tests))
}

func TestGetTechnique(t *testing.T) {
	ar, err := newRunner(testFolder)
	require.NoError(t, err)
	tech, err := ar.GetTechnique("T9999")
	require.NoError(t, err)
	assert.Equal(t, "T9999", tech.ID)
	assert.Equal(t, 4, len(tech.AtomicTests))
	assert.Equal(t, "Test1", tech.AtomicTests[0].Name)
}

func TestGetTestByIDAndIndex(t *testing.T) {
	ar, err := newRunner(testFolder)
	require.NoError(t, err)
	test, err := ar.GetTestByIDAndIndex("T9999", 0)
	require.NoError(t, err)
	assert.Equal(t, "Test1", test.Name)
}

func TestGetTestByIDAndName(t *testing.T) {
	ar, err := newRunner(testFolder)
	require.NoError(t, err)
	test, err := ar.GetTestByIDAndName("T9999", "Test1")
	require.NoError(t, err)
	assert.Equal(t, "Test1", test.Name)
}

func TestFilter_AtomicTests(t *testing.T) {
	var test = []struct {
		config FilterConfig
		want   [2]int
	}{
		{
			config: newFilter("", []string{"T9999"}, true),
			want:   [2]int{1, 4},
		}, {
			config: newFilter("", []string{"T9999"}, false),
			want:   [2]int{1, 3},
		}, {
			config: newFilter("macos", []string{"T9999"}, false),
			want:   [2]int{1, 2},
		}, {
			config: newFilter("macos", []string{"T9999"}, true),
			want:   [2]int{1, 3},
		},
	}
	ar, err := newRunner(testFolder)
	require.NoError(t, err)
	for id, tt := range test {
		t.Run(fmt.Sprintf("TestFilter_AtomicTests %d", id), func(t *testing.T) {
			out := ar.Filter(&tt.config)
			assert.Equal(t, tt.want[0], len(out))
			assert.Equal(t, tt.want[1], len(out[0].AtomicTests))
		})
	}
}

func newFilter(platform string, techniques []string, includeManual bool) FilterConfig {
	return FilterConfig{
		Platform:      platform,
		Techniques:    techniques,
		IncludeManual: includeManual,
	}
}

func TestFilter_Techniques(t *testing.T) {
	var empty []string
	var test = []struct {
		config FilterConfig
		want   int
	}{
		{
			config: newFilter("", []string{"TXXXX"}, false),
			want:   0,
		}, {
			config: newFilter("linux", []string{"T9999"}, false),
			want:   1,
		}, {
			config: newFilter("fake", []string{"T9999"}, false),
			want:   0,
		}, {
			config: newFilter("", empty, false),
			want:   3,
		}, {
			config: newFilter("", empty, true),
			want:   4,
		},
	}
	ar, err := newRunner(testFolder)
	require.NoError(t, err)
	for id, tt := range test {
		t.Run(fmt.Sprintf("TestFilter_Techniques %d", id), func(t *testing.T) {
			out := ar.Filter(&tt.config)
			assert.Equal(t, tt.want, len(out))
		})
	}
}

func TestFilter_All(t *testing.T) {
	ar, err := newRunner(testFolder)
	require.NoError(t, err)
	allTest := ar.GetAllTechniques()
	// this operation should return the entire set of atomic tests without removing anything
	filtered := ar.Filter(&FilterConfig{
		IncludeManual: true,
	})
	sort.Slice(allTest, func(i, j int) bool {
		return allTest[i].ID < allTest[j].ID
	})
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ID < filtered[j].ID
	})
	assert.Equal(t, len(allTest), len(filtered))
	for i := range allTest {
		assert.Equal(t, allTest[i].AtomicTests, filtered[i].AtomicTests)
	}
}

func getMockTest() (*art.Test, map[string]string) {
	defaultArgs := make(map[string]art.Argument)
	defaultArgs["prereq"] = art.Argument{
		Default: "prereq-default",
	}
	defaultArgs["getprereq"] = art.Argument{
		Default: "getprereq-default",
	}
	defaultArgs["command"] = art.Argument{
		Default: "command-default",
	}
	defaultArgs["cleanup"] = art.Argument{
		Default: "cleanup-default",
	}
	userSuppliedArgs := make(map[string]string)
	userSuppliedArgs["command"] = "command-user"
	userSuppliedArgs["cleanup"] = "cleanup-user"
	userSuppliedArgs["prereq"] = "prereq-user"

	atomicTest := art.Test{
		TechniqueID:            "T9999",
		Name:                   "Test",
		Description:            "Test",
		SupportedPlatforms:     []string{getCurrentPlatform()},
		InputArguments:         defaultArgs,
		DependencyExecutorName: "command_prompt",
		Dependencies: []art.Dependency{
			{
				Description:      "test dependency 1",
				PrereqCommand:    "echo #{prereq}\nexit 123",
				GetPrereqCommand: "echo #{getprereq}",
			},
			{
				Description:      "test dependency 2",
				PrereqCommand:    "echo #{prereq}\nexit 0",
				GetPrereqCommand: "echo #{getprereq}",
			},
		},
		Executor: art.Executor{
			Name:              "command_prompt",
			ElevationRequired: false,
			Command:           "echo #{command}",
			CleanupCommand:    "echo #{cleanup}",
		},
	}
	return &atomicTest, userSuppliedArgs
}

func TestBuildTest(t *testing.T) {
	ar := Runner{}
	at, args := getMockTest()
	bt, err := ar.BuildTest(at, args)
	require.NoError(t, err)

	assert.Equal(t, "echo command-user", bt.AtomicTestCommands)
	assert.Equal(t, "echo cleanup-user", bt.CleanupCommands)
	require.NotNil(t, 1, bt.DependencyInfo)
	assert.Equal(t, 2, len(bt.DependencyInfo.Dependencies))
	assert.Equal(t, "echo prereq-user\nexit 123", bt.DependencyInfo.Dependencies[0].PreReqCmds)
	assert.Equal(t, "echo getprereq-default", bt.DependencyInfo.Dependencies[0].GetPreReqCmds)
}

func TestProcessYamlFolder(t *testing.T) {
	ar := Runner{AtomicsFolder: filepath.Join(testFolder, "T9999")}
	tests := ar.processYAMLFolder()
	assert.Equal(t, 1, len(tests))
	assert.NotNil(t, tests[0].Path)
	assert.Equal(t, 4, len(tests[0].AtomicTests))
	assert.Equal(t, 2, len(tests[0].AtomicTests[0].SupportedPlatforms))
	assert.Equal(t, "PathToAtomicsFolder/src/test.txt",
		tests[0].AtomicTests[1].InputArguments["file_name"].Default)
	assert.Equal(t, "", tests[0].AtomicTests[2].Executor.Name)
}
