package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseString(t *testing.T) {
	testYaml := `---
attack_technique: TODO
display_name: TODO
path: test

atomic_tests:
- name: TODO
  description: |
    TODO

  supported_platforms:
    - windows
    - macos
    - linux

  input_arguments:
    output_file:
      description: TODO
      type: todo
      default: TODO

  dependency_executor_name: powershell # (optional) The executor for the prereq commands, defaults to the same executor used by the attack commands
  dependencies: # (optional)
    - description: |
        TODO
      prereq_command: | # commands to check if prerequisites for running this test are met. For the "command_prompt" executor, if any command returns a non-zero exit code, the pre-requisites are not met. For the "powershell" executor, all commands are run as a script block and the script block must return 0 for success.
        TODO
      get_prereq_command: | # commands to meet this prerequisite or a message describing how to meet this prereq
        TODO
    - description: |
        TODO2
      prereq_command: | # commands to check if prerequisites for running this test are met. For the "command_prompt" executor, if any command returns a non-zero exit code, the pre-requisites are not met. For the "powershell" executor, all commands are run as a script block and the script block must return 0 for success.
        TODO2
      get_prereq_command: | # commands to meet this prerequisite or a message describing how to meet this prereq
        TODO2
  executor:
    name: command_prompt
    elevation_required: true # indicates whether command must be run with admin privileges. If the elevation_required attribute is not defined, the value is assumed to be false
    command: | # these are the actual attack commands, at least one command must be provided
      TODO
    cleanup_command: | # you can remove the cleanup_command section if there are no cleanup commands
      TODO`

	technique, err := parse([]byte(testYaml))
	assert.NoError(t, err)
	assert.Equal(t, "", technique.Path)
	assert.Equal(t, "TODO", technique.DisplayName)
	assert.Equal(t, "TODO", technique.ID)
	assert.Equal(t, 1, len(technique.AtomicTests))
	assert.Equal(t, "TODO", technique.AtomicTests[0].Name)
	assert.Equal(t, "TODO\n", technique.AtomicTests[0].Description)
	assert.Equal(t, []string{"windows", "macos", "linux"}, technique.AtomicTests[0].SupportedPlatforms)
	assert.Equal(t, 2, len(technique.AtomicTests[0].Dependencies))
	assert.Equal(t, "TODO\n", technique.AtomicTests[0].Dependencies[0].GetPrereqCommand)
}

func TestManualExecutor(t *testing.T) {
	testYaml := `---
attack_technique: T9999
display_name: TestData

atomic_tests:
  - name: Test1
    description: |
      Test Linux & Mac
    supported_platforms:
      - macos
      - linux
    input_arguments:
      file_name:
        description: filename
        type: Path
        default: PathToAtomicsFolder/src/test.txt

    executor:
      name: command_prompt
      elevation_required: false
      command: |
        cat ${file_name}

  - name: Test2
    description: |
      Test Windows
    supported_platforms:
      - windows
    input_arguments:
      file_name:
        description: filename
        type: Path
        default: PathToAtomicsFolder/src/test.txt
    executor:
      name: powershell
      elevation_required: false
      command: |
        cat ${file_name}
        cat ${file_name}

  - name: Test3
    description: |
      Test without executor
    supported_platforms:
      - macos

  - name: Test4
    description: |
      Test without executor
    supported_platforms:
      - macos
    executor:
      name: manual
      steps: do nothing`
	technique, err := parse([]byte(testYaml))
	assert.NoError(t, err)
	assert.Equal(t, 4, len(technique.AtomicTests))
	assert.Equal(t, "Test1", technique.AtomicTests[0].Name)
	assert.Equal(t, "", technique.AtomicTests[2].Executor.Name)
}
