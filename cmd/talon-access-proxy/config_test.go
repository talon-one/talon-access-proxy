package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/hjson/hjson-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func createConfig(t *testing.T, v interface{}) string {
	tmpfile, err := ioutil.TempFile("", "tap-test-")
	require.NoError(t, err)
	buffer, err := hjson.Marshal(v)
	require.NoError(t, err)
	_, err = tmpfile.Write(buffer)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())
	return tmpfile.Name()
}

func TestReadConfigs(t *testing.T) {
	t.Run("Invalid Config File", func(t *testing.T) {
		tmpfile, err := ioutil.TempFile("", "tap-test-")
		require.NoError(t, err)
		tmpfile.WriteString("{")
		require.NoError(t, tmpfile.Close())
		os.Setenv("APP_CONFIG", tmpfile.Name())
		defer os.Remove(tmpfile.Name())
		defer os.Unsetenv("APP_CONFIG")

		_, err = readConfigs()
		require.Error(t, err)
	})
	t.Run("Invalid Address", func(t *testing.T) {
		file := createConfig(t, map[string]interface{}{
			"Address": "127.0.0.1",
		})
		os.Setenv("APP_CONFIG", file)
		defer os.Remove(file)
		defer os.Unsetenv("APP_CONFIG")

		_, err := readConfigs()
		require.Error(t, err)
	})
	t.Run("MaxConnections", func(t *testing.T) {
		t.Run("No Setting Should Resolve to Default Value", func(t *testing.T) {
			file := createConfig(t, map[string]interface{}{})
			os.Setenv("APP_CONFIG", file)
			defer os.Remove(file)
			defer os.Unsetenv("APP_CONFIG")
			configs, err := readConfigs()
			require.NoError(t, err)
			require.Equal(t, 100, configs[0].Config.MaxConnections)
		})
		t.Run("Setted Value", func(t *testing.T) {
			file := createConfig(t, map[string]interface{}{
				"MaxConnections": 200,
			})
			os.Setenv("APP_CONFIG", file)
			defer os.Remove(file)
			defer os.Unsetenv("APP_CONFIG")
			configs, err := readConfigs()
			require.NoError(t, err)
			require.Equal(t, 200, configs[0].Config.MaxConnections)
		})
	})
	t.Run("Multiple Config Files", func(t *testing.T) {
		file := createConfig(t, []map[string]interface{}{
			map[string]interface{}{
				"Address": "127.0.0.1:8000",
			},
			map[string]interface{}{
				"Address": "127.0.0.1:8001",
			},
		})
		os.Setenv("APP_CONFIG", file)
		defer os.Remove(file)
		defer os.Unsetenv("APP_CONFIG")
		configs, err := readConfigs()
		require.NoError(t, err)
		require.Equal(t, "127.0.0.1:8000", configs[0].Address)
		require.Equal(t, "127.0.0.1:8001", configs[1].Address)
	})
}

func TestDebugLevel(t *testing.T) {
	t.Run("Debug", func(t *testing.T) {
		t.Run("No Setting Should Resolve to Default Value", func(t *testing.T) {
			file := createConfig(t, map[string]interface{}{})
			os.Setenv("APP_CONFIG", file)
			defer os.Remove(file)
			defer os.Unsetenv("APP_CONFIG")
			configs, err := readConfigs()
			require.NoError(t, err)
			require.Equal(t, false, configs[0].Logger.Core().Enabled(zapcore.DebugLevel))
		})
		t.Run("Setted Value", func(t *testing.T) {
			file := createConfig(t, map[string]interface{}{})
			os.Setenv("APP_CONFIG", file)
			os.Setenv("DEBUG", "1")
			defer os.Remove(file)
			defer os.Unsetenv("APP_CONFIG")
			configs, err := readConfigs()
			require.NoError(t, err)
			require.Equal(t, true, configs[0].Logger.Core().Enabled(zapcore.DebugLevel))
		})
	})
}
