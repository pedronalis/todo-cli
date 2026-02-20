package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"todo-cli/model"
)

const maxRotatingBackups = 10

var errNoValidBackup = errors.New("no valid backup found")

// Load reads app state from a JSON file.
// If file does not exist, it returns an initialized empty state.
func Load(path string) (model.AppState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return model.NewState(), nil
		}
		return model.AppState{}, err
	}
	return decodeState(data)
}

// LoadWithRecovery loads state and tries automatic recovery when the main JSON is corrupted.
// It returns an optional status message to be shown to the user.
func LoadWithRecovery(path string) (model.AppState, string, error) {
	state, err := Load(path)
	if err == nil {
		return state, "", nil
	}
	if !isCorruptStateError(err) {
		return model.AppState{}, "", err
	}

	corruptPath, moveErr := moveCorruptFile(path)
	if moveErr != nil {
		return model.AppState{}, "", fmt.Errorf("falha ao mover arquivo corrompido: %w", moveErr)
	}

	recoveredState, backupPath, backupErr := loadLatestValidBackup(path)
	if backupErr == nil {
		if err := Save(path, recoveredState); err != nil {
			return model.AppState{}, "", fmt.Errorf("falha ao restaurar backup: %w", err)
		}
		msg := fmt.Sprintf("Estado corrompido recuperado de %s", filepath.Base(backupPath))
		if corruptPath != "" {
			msg += fmt.Sprintf(" (arquivo ruim movido para %s)", filepath.Base(corruptPath))
		}
		return recoveredState, msg, nil
	}
	if !errors.Is(backupErr, errNoValidBackup) {
		return model.AppState{}, "", fmt.Errorf("falha ao inspecionar backups: %w", backupErr)
	}

	empty := model.NewState()
	if err := Save(path, empty); err != nil {
		return model.AppState{}, "", fmt.Errorf("falha ao inicializar novo estado após corrupção: %w", err)
	}
	msg := "Estado corrompido sem backup válido; iniciado com estado vazio"
	if corruptPath != "" {
		msg += fmt.Sprintf(" (arquivo ruim movido para %s)", filepath.Base(corruptPath))
	}
	return empty, msg, nil
}

// Save writes app state to path as JSON.
func Save(path string, state model.AppState) error {
	return writeJSON(path, state)
}

// Autosave writes safely using temporary file + atomic rename.
// It also stores a latest backup (.bak) and a rotating timestamped backup set.
func Autosave(path string, state model.AppState) error {
	if err := ensureDir(path); err != nil {
		return err
	}

	if err := backup(path); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(state); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, path)
}

func decodeState(data []byte) (model.AppState, error) {
	var state model.AppState
	if err := json.Unmarshal(data, &state); err != nil {
		return model.AppState{}, err
	}

	if state.Lists == nil {
		state.Lists = []model.List{}
	}
	if state.Tasks == nil {
		state.Tasks = []model.Task{}
	}
	if state.ArchivedCompleted == nil {
		state.ArchivedCompleted = []model.ArchivedCompletedTask{}
	}
	if state.Filter == "" {
		state.Filter = model.FilterAll
	}
	if state.Metadata.Version == 0 {
		state.Metadata.Version = 1
	}
	if strings.TrimSpace(state.Metadata.Session.Focus) == "" {
		state.Metadata.Session.Focus = model.SessionFocusLists
	}

	return state, nil
}

func writeJSON(path string, state model.AppState) error {
	if err := ensureDir(path); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0o755)
}

func backup(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	if err := os.WriteFile(path+".bak", data, 0o644); err != nil {
		return err
	}

	timestamp := time.Now().UTC().Format("20060102-150405.000000000")
	rotatingPath := fmt.Sprintf("%s.bak.%s", path, timestamp)
	if err := os.WriteFile(rotatingPath, data, 0o644); err != nil {
		return err
	}

	return pruneRotatingBackups(path)
}

func pruneRotatingBackups(path string) error {
	files, err := filepath.Glob(path + ".bak.*")
	if err != nil {
		return err
	}
	if len(files) <= maxRotatingBackups {
		return nil
	}

	sort.Strings(files)
	toDelete := files[:len(files)-maxRotatingBackups]
	for _, old := range toDelete {
		if err := os.Remove(old); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func loadLatestValidBackup(path string) (model.AppState, string, error) {
	candidates := make([]string, 0, 12)
	latest := path + ".bak"
	if _, err := os.Stat(latest); err == nil {
		candidates = append(candidates, latest)
	}
	rotating, err := filepath.Glob(path + ".bak.*")
	if err != nil {
		return model.AppState{}, "", err
	}
	candidates = append(candidates, rotating...)
	if len(candidates) == 0 {
		return model.AppState{}, "", errNoValidBackup
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		iInfo, iErr := os.Stat(candidates[i])
		jInfo, jErr := os.Stat(candidates[j])
		if iErr != nil || jErr != nil {
			return candidates[i] > candidates[j]
		}
		return iInfo.ModTime().After(jInfo.ModTime())
	})

	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		state, err := decodeState(data)
		if err != nil {
			continue
		}
		return state, candidate, nil
	}

	return model.AppState{}, "", errNoValidBackup
}

func moveCorruptFile(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	timestamp := time.Now().UTC().Format("20060102-150405")
	corruptName := fmt.Sprintf("%s.corrupt-%s%s", name, timestamp, ext)
	corruptPath := filepath.Join(filepath.Dir(path), corruptName)
	if err := os.Rename(path, corruptPath); err != nil {
		return "", err
	}
	return corruptPath, nil
}

func isCorruptStateError(err error) bool {
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return true
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return true
	}
	return errors.Is(err, io.ErrUnexpectedEOF)
}
