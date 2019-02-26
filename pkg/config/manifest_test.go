package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadManifest_filepath(t *testing.T)  {
	c := NewTestConfig(t)
	c.TestContext.AddTestFile("testdata/simple.porter.yaml", Name)

	m, err := c.ReadManifest(Name)
	require.NoError(t, err)
	assert.Equal(t, "hello" , m.Name)
	assert.Equal(t, Name, m.path)
}

func TestReadManifest_url(t *testing.T)  {
	c := NewTestConfig(t)

	m, err := c.ReadManifest("https://raw.githubusercontent.com/deislabs/porter/master/pkg/config/testdata/porter.yaml")
	require.NoError(t, err)
	assert.Equal(t, "https://raw.githubusercontent.com/deislabs/porter/master/pkg/config/testdata/porter.yaml", m.path)
}

func TestLoadManifest(t *testing.T) {
	c := NewTestConfig(t)
	c.SetupPorterHome()

	c.TestContext.AddTestFile("testdata/simple.porter.yaml", Name)

	require.NoError(t, c.LoadManifest())

	assert.NotNil(t, c.Manifest)
	assert.Equal(t, []string{"exec"}, c.Manifest.Mixins)
	assert.Len(t, c.Manifest.Install, 1)

	installStep := c.Manifest.Install[0]
	description, _ := installStep.GetDescription()
	assert.NotNil(t, description)

	mixin := installStep.GetMixinName()
	assert.Equal(t, "exec", mixin)
}

func TestLoadManifestWithDependencies(t *testing.T) {
	c := NewTestConfig(t)
	c.SetupPorterHome()

	c.TestContext.AddTestFile("testdata/porter.yaml", Name)
	c.TestContext.AddTestDirectory("testdata/bundles", "bundles")

	require.NoError(t, c.LoadManifest())

	assert.NotNil(t, c.Manifest)
	assert.Equal(t, []string{"helm", "exec"}, c.Manifest.Mixins)
	assert.Len(t, c.Manifest.Install, 2)

	installStep := c.Manifest.Install[0]
	description, _ := installStep.GetDescription()
	assert.NotNil(t, description)

	mixin := installStep.GetMixinName()
	assert.Equal(t, "helm", mixin)
}

func TestConfig_LoadManifest_BundleDependencyNotInstalled(t *testing.T) {
	c := NewTestConfig(t)

	c.TestContext.AddTestFile("testdata/missingdep.porter.yaml", Name)

	err := c.LoadManifest()
	require.Errorf(t, err, "bundle missingdep not installed in PORTER_HOME")
}

func TestAction_Validate_RequireMixinDeclaration(t *testing.T) {
	c := NewTestConfig(t)
	c.SetupPorterHome()

	c.TestContext.AddTestFile("testdata/simple.porter.yaml", Name)

	err := c.LoadManifest()
	require.NoError(t, err)

	// Sabotage!
	c.Manifest.Mixins = []string{}

	err = c.Manifest.Install.Validate(c.Manifest)
	assert.EqualError(t, err, "mixin (exec) was not declared")
}

func TestAction_Validate_RequireMixinData(t *testing.T) {
	c := NewTestConfig(t)
	c.SetupPorterHome()

	c.TestContext.AddTestFile("testdata/simple.porter.yaml", Name)

	err := c.LoadManifest()
	require.NoError(t, err)

	// Sabotage!
	c.Manifest.Install[0].Data = nil

	err = c.Manifest.Install.Validate(c.Manifest)
	assert.EqualError(t, err, "no mixin specified")
}

func TestAction_Validate_RequireSingleMixinData(t *testing.T) {
	c := NewTestConfig(t)
	c.SetupPorterHome()

	c.TestContext.AddTestFile("testdata/simple.porter.yaml", Name)

	err := c.LoadManifest()
	require.NoError(t, err)

	// Sabotage!
	c.Manifest.Install[0].Data["rando-mixin"] = ""

	err = c.Manifest.Install.Validate(c.Manifest)
	assert.EqualError(t, err, "more than one mixin specified")
}

func TestResolveMapParam(t *testing.T) {
	m := &Manifest{
		Parameters: []ParameterDefinition{
			{
				Name: "person",
			},
		},
	}

	os.Setenv("PERSON", "Ralpha")
	s := &Step{
		Data: map[string]interface{}{
			"description": "a test step",
			"Parameters": map[string]interface{}{
				"Thing": map[string]interface{}{
					"source": "bundle.parameters.person",
				},
			},
		},
	}

	err := m.ResolveStep(s)
	require.NoError(t, err)
	pms, ok := s.Data["Parameters"].(map[string]interface{})
	assert.True(t, ok)
	val, ok := pms["Thing"].(string)
	assert.True(t, ok)
	assert.Equal(t, "Ralpha", val)
}

func TestResolveMapParamUnknown(t *testing.T) {

	m := &Manifest{
		Parameters: []ParameterDefinition{},
	}

	s := &Step{
		Data: map[string]interface{}{
			"description": "a test step",
			"Parameters": map[string]interface{}{
				"Thing": map[string]interface{}{
					"source": "bundle.parameters.person",
				},
			},
		},
	}

	err := m.ResolveStep(s)
	require.Error(t, err)
	assert.Equal(t, "unable to set value for Thing: no value found for source specification: bundle.parameters.person", err.Error())
}

func TestResolveArrayUnknown(t *testing.T) {
	m := &Manifest{
		Parameters: []ParameterDefinition{
			{
				Name: "name",
			},
		},
	}

	s := &Step{
		Data: map[string]interface{}{
			"description": "a test step",
			"Arguments": []string{
				"source: bundle.parameters.person",
			},
		},
	}

	err := m.ResolveStep(s)
	require.Error(t, err)
	assert.Equal(t, "unable to source value: no value found for source specification: bundle.parameters.person", err.Error())
}

func TestResolveArray(t *testing.T) {
	m := &Manifest{
		Parameters: []ParameterDefinition{
			{
				Name: "person",
			},
		},
	}

	os.Setenv("PERSON", "Ralpha")
	s := &Step{
		Data: map[string]interface{}{
			"description": "a test step",
			"Arguments": []string{
				"source: bundle.parameters.person",
			},
		},
	}

	err := m.ResolveStep(s)
	require.NoError(t, err)
	args, ok := s.Data["Arguments"].([]string)
	assert.True(t, ok)
	assert.Equal(t, "Ralpha", args[0])
}

func TestResolveInMainDict(t *testing.T) {
	c := NewTestConfig(t)
	c.SetupPorterHome()

	c.TestContext.AddTestFile("testdata/param-test-in-block.yaml", Name)

	require.NoError(t, c.LoadManifest())

	installStep := c.Manifest.Install[0]

	os.Setenv("COMMAND", "echo hello world")
	err := c.Manifest.ResolveStep(installStep)
	assert.NoError(t, err)

	assert.NotNil(t, installStep.Data)
	t.Logf("install data %v", installStep.Data)
	exec := installStep.Data["exec"].(map[interface{}]interface{})
	assert.NotNil(t, exec)
	command := exec["command"].(interface{})
	assert.NotNil(t, command)
	cmdVal, ok := command.(string)
	assert.True(t, ok)
	assert.Equal(t, "echo hello world", cmdVal)
}

func TestResolveSliceWithAMap(t *testing.T) {
	c := NewTestConfig(t)
	c.SetupPorterHome()

	c.TestContext.AddTestFile("testdata/slice-test.yaml", Name)

	require.NoError(t, c.LoadManifest())

	installStep := c.Manifest.Install[0]

	os.Setenv("COMMAND", "echo hello world")
	err := c.Manifest.ResolveStep(installStep)
	assert.NoError(t, err)

	assert.NotNil(t, installStep.Data)
	t.Logf("install data %v", installStep.Data)
	exec := installStep.Data["exec"].(map[interface{}]interface{})
	assert.NotNil(t, exec)
	args := exec["arguments"].([]interface{})
	assert.Len(t, args, 2)
	assert.Equal(t, "echo hello world", args[1])
	assert.NotNil(t, args)
}

func TestDependency_Validate_NameRequired(t *testing.T) {
	c := NewTestConfig(t)
	c.SetupPorterHome()

	c.TestContext.AddTestFile("testdata/porter.yaml", Name)
	c.TestContext.AddTestDirectory("testdata/bundles", "bundles")

	err := c.LoadManifest()
	require.NoError(t, err)

	// Sabotage!
	c.Manifest.Dependencies[0].Name = ""

	err = c.Manifest.Dependencies[0].Validate()
	assert.EqualError(t, err, "dependency name is required")
}

func TestManifest_MergeDependency(t *testing.T) {
	m := &Manifest{
		Mixins: []string{"helm"},
		Install: Steps{
			&Step{Data: map[string]interface{}{"helm": map[interface{}]interface{}{"description": "install wordpress"}}},
		},
		Upgrade: Steps{
			&Step{Data: map[string]interface{}{"helm": map[interface{}]interface{}{"description": "upgrade wordpress"}}},
		},
		Uninstall: Steps{
			&Step{Data: map[string]interface{}{"helm": map[interface{}]interface{}{"description": "uninstall wordpress"}}},
		},
	}

	dep := &Dependency{
		m: &Manifest{
			Mixins: []string{"exec", "helm"},
			Install: Steps{
				&Step{Data: map[string]interface{}{"helm": map[interface{}]interface{}{"description": "install mysql"}}},
			},
			Upgrade: Steps{
				&Step{Data: map[string]interface{}{"helm": map[interface{}]interface{}{"description": "upgrade mysql"}}},
			},
			Uninstall: Steps{
				&Step{Data: map[string]interface{}{"helm": map[interface{}]interface{}{"description": "uninstall mysql"}}},
			},
			Credentials: []CredentialDefinition{
				{Name: "kubeconfig", Path: "/root/.kube/config"},
			},
		},
	}

	err := m.MergeDependency(dep)
	require.NoError(t, err)

	assert.Equal(t, []string{"exec", "helm"}, m.Mixins)

	assert.Len(t, m.Install, 2)
	description, _ := m.Install[0].GetDescription()
	assert.Equal(t, "install mysql", description)
	description, _ = m.Install[1].GetDescription()
	assert.Equal(t, "install wordpress", description)

	assert.Len(t, m.Upgrade, 2)
	description, _ = m.Upgrade[0].GetDescription()
	assert.Equal(t, "upgrade mysql", description)
	description, _ = m.Upgrade[1].GetDescription()
	assert.Equal(t, "upgrade wordpress", description)

	assert.Len(t, m.Uninstall, 2)
	description, _ = m.Uninstall[0].GetDescription()
	assert.Equal(t, "uninstall wordpress", description)
	description, _ = m.Uninstall[1].GetDescription()
	assert.Equal(t, "uninstall mysql", description)

	assert.Len(t, m.Credentials, 1)
}

func TestMergeCredentials(t *testing.T) {
	testcases := []struct {
		name               string
		c1, c2, wantResult CredentialDefinition
		wantError          string
	}{
		{
			name:       "combine path and environment variable",
			c1:         CredentialDefinition{Name: "foo", Path: "p1"},
			c2:         CredentialDefinition{Name: "foo", EnvironmentVariable: "v2"},
			wantResult: CredentialDefinition{Name: "foo", Path: "p1", EnvironmentVariable: "v2"},
		},
		{
			name:       "same path",
			c1:         CredentialDefinition{Name: "foo", Path: "p"},
			c2:         CredentialDefinition{Name: "foo", Path: "p"},
			wantResult: CredentialDefinition{Name: "foo", Path: "p"},
		},
		{
			name:      "conflicting path",
			c1:        CredentialDefinition{Name: "foo", Path: "p1"},
			c2:        CredentialDefinition{Name: "foo", Path: "p2"},
			wantError: "cannot merge credential foo: conflict on path",
		},
		{
			name:       "same environment variable",
			c1:         CredentialDefinition{Name: "foo", EnvironmentVariable: "v"},
			c2:         CredentialDefinition{Name: "foo", EnvironmentVariable: "v"},
			wantResult: CredentialDefinition{Name: "foo", EnvironmentVariable: "v"},
		},
		{
			name:      "conflicting environment variable",
			c1:        CredentialDefinition{Name: "foo", EnvironmentVariable: "v1"},
			c2:        CredentialDefinition{Name: "foo", EnvironmentVariable: "v2"},
			wantError: "cannot merge credential foo: conflict on environment variable",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := mergeCredentials(tc.c1, tc.c2)

			if tc.wantError == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.wantResult, result)
			} else {
				require.Contains(t, err.Error(), tc.wantError)
			}
		})
	}

}

func TestManifest_ApplyBundleOutputs(t *testing.T) {
	c := NewTestConfig(t)
	c.SetupPorterHome()

	c.TestContext.AddTestFile("testdata/simple.porter.yaml", Name)

	require.NoError(t, c.LoadManifest())

	depStep := c.Manifest.Install[0]
	err := c.Manifest.ApplyOutputs(depStep, []string{"foo=bar"})
	require.NoError(t, err)

	assert.Contains(t, c.Manifest.outputs, "foo")
	assert.Equal(t, "bar", c.Manifest.outputs["foo"])
}

func TestManifest_ApplyDependencyOutputs(t *testing.T) {
	testcases := []struct {
		name        string
		rawOutputs  []string
		wantOutputs map[string]string
		wantError   string
	}{
		{
			name:        "happy path",
			rawOutputs:  []string{"host=localhost"},
			wantOutputs: map[string]string{"host": "localhost"},
		},
		{
			name:        "value with equals sign",
			rawOutputs:  []string{"cert=abc123==="},
			wantOutputs: map[string]string{"cert": "abc123==="},
		},
		{
			name:       "missing equals sign",
			rawOutputs: []string{"foo"},
			wantError:  "invalid output assignment",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewTestConfig(t)
			c.SetupPorterHome()

			c.TestContext.AddTestFile("testdata/porter.yaml", Name)
			c.TestContext.AddTestDirectory("testdata/bundles", "bundles")

			require.NoError(t, c.LoadManifest())

			depStep := c.Manifest.Install[0]
			err := c.Manifest.ApplyOutputs(depStep, tc.rawOutputs)
			if tc.wantError == "" {
				require.NoError(t, err)
			} else {
				require.Contains(t, err.Error(), tc.wantError)
				return
			}

			depM := c.Manifest.Dependencies[0].m
			for wantKey, wantValue := range tc.wantOutputs {
				assert.Contains(t, depM.outputs, wantKey)
				assert.Equal(t, wantValue, depM.outputs[wantKey])
			}
		})
	}
}

func TestManifest_resolveSource(t *testing.T) {
	testcases := []struct {
		name       string
		outputs    map[string]string
		source     string
		wantResult interface{}
		wantError  string
	}{
		{
			name:       "happy path",
			outputs:    map[string]string{"foo": "bar"},
			source:     "bundle.outputs.foo",
			wantResult: "bar",
		},
		{
			name:      "missing output",
			outputs:   map[string]string{"foo": "bar"},
			source:    "bundle.outputs.missing",
			wantError: "no value found for source specification: bundle.outputs.missing",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			m := &Manifest{
				outputs: tc.outputs,
			}

			result, err := m.resolveValue(tc.source)
			if tc.wantError == "" {
				require.NoError(t, err)
			} else {
				require.Contains(t, err.Error(), tc.wantError)
				return
			}

			assert.Equal(t, tc.wantResult, result)
		})
	}
}

func TestManifest_MergeParameters(t *testing.T) {
	dep := &Dependency{
		Name:       "mysql",
		Parameters: map[string]string{"database": "wordpress"},
		m: &Manifest{
			Name: "mysql",
			Parameters: []ParameterDefinition{
				{Name: "database"},
			},
		},
	}
	m := &Manifest{
		Name:         "wordpress",
		Dependencies: []*Dependency{dep},
	}

	err := m.MergeParameters(dep)
	require.NoError(t, err)

	require.Len(t, m.Parameters, 1)
	assert.Equal(t, "wordpress", m.Parameters[0].DefaultValue)
}
