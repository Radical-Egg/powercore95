package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"gopkg.in/ini.v1"
)

var templates = template.Must(template.New("").Funcs(template.FuncMap{
	"GetSandboxConfig": GetSandboxConfig,
	"toLower":          strings.ToLower,
}).ParseGlob("templates/*.html"))

var SANDBOX_INI_DEFAULTS = "./data/Sandbox.ini.default"
var SANDBOX_JSON_DEFAULT = "./data/sandbox.json"

func Json2Struct() *Config {
	file, err := os.ReadFile(SANDBOX_JSON_DEFAULT)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(file, &config); err != nil {
		log.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	return &config
}

func Ini2Struct() (*Sandbox, error) {
	cfg, err := ini.Load(os.Getenv("SANDBOX_INI_PATH"))

	if err != nil {
		log.Fatalf("Failed to read INI file: %v", err)
	}

	var sbx Sandbox
	err = cfg.Section("SandboxSettings").MapTo(&sbx)

	if err != nil {
		return nil, err
	}
	return &sbx, nil
}

func GetSandboxConfig(key string, obj Sandbox) (interface{}, error) {
	val := reflect.ValueOf(obj)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("provided value is not a struct or pointer to a struct")
	}

	field := val.FieldByName(key)

	if !field.IsValid() {
		fmt.Println(field)
		return nil, fmt.Errorf("field %q does not exist in struct", key)
	}

	return field.Interface(), nil
}

func writeIni(sbx *Sandbox, filename string) error {
	cfg := ini.Empty()

	sandboxSection, err := cfg.NewSection("SandboxSettings")
	if err != nil {
		fmt.Println("Error creating section:", err)
		return err
	}

	err = sandboxSection.ReflectFrom(sbx)

	if err != nil {
		return err
	}

	err = cfg.SaveTo(filename)
	if err != nil {
		return fmt.Errorf("failed to write INI file: %w", err)
	}

	return nil
}

func setEnvironmentVariables() {
	_, ok := os.LookupEnv("SANDBOX_INI_PATH")

	if !ok {
		os.Setenv("SANDBOX_INI_PATH",
			"/gamedata/Config/WindowsServer/Sandbox.ini")
	}

	_, ok = os.LookupEnv("CONTAINER_NAME")

	if !ok {
		os.Setenv("CONTAINER_NAME", "abiotic")
	}

	_, ok = os.LookupEnv("SERVER_NAME")

	if !ok {
		os.Setenv("SERVER_NAME", "Mobo Blob")
	}
}

func ContainerAction(action string, cli *client.Client, containerID string) error {
	switch action {
	case "restart":
		timeout := 10
		return cli.ContainerRestart(context.Background(), containerID, container.StopOptions{
			Timeout: &timeout,
		})
	default:
		return fmt.Errorf("i do not understand the command %s", action)
	}
}
func copySandboxDefaults(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

func main() {
	setEnvironmentVariables()
	if _, err := os.Stat(os.Getenv("SANDBOX_INI_PATH")); errors.Is(err, os.ErrNotExist) {
		dirPath := filepath.Dir(os.Getenv("SANDBOX_INI_PATH"))

		err := os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
		copySandboxDefaults(SANDBOX_INI_DEFAULTS, os.Getenv("SANDBOX_INI_PATH"))
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/write-ini", submitHandler)
	http.HandleFunc("/restart-container", restartContainerHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	log.Println("Server running at http://localhost:9090")
	log.Fatal(http.ListenAndServe(":9090", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	sbx, err := Ini2Struct()

	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}

	defaults := Json2Struct()

	data := struct {
		Title          string
		DefaultConfigs *Config
		Current        *Sandbox
		GridContainers []string
	}{
		Title:          os.Getenv("SERVER_NAME"),
		DefaultConfigs: defaults,
		Current:        sbx,
		GridContainers: []string{"Player", "World", "Enemy"},
	}
	err = templates.ExecuteTemplate(w, "base.html", data)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var sbx Sandbox

	err := json.NewDecoder(r.Body).Decode(&sbx)

	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return
	}
	err = writeIni(&sbx, os.Getenv("SANDBOX_INI_PATH"))

	if err != nil {
		http.Error(w, "Unable to write INI file", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func restartContainerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	err = ContainerAction("restart", cli, os.Getenv("CONTAINER_NAME"))

	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

type RangeInput struct {
	Key       string  `json:"key"`
	Default   float64 `json:"default"`
	InputType string  `json:"inputType"`
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
	Step      float64 `json:"step"`
}

type ToggleInput struct {
	Key       string `json:"key"`
	InputType string `json:"inputType"`
	Default   bool   `json:"default"`
}

type Config struct {
	RangeInputs  []RangeInput  `json:"rangeInputs"`
	ToggleInputs []ToggleInput `json:"toggleInputs"`
}

type Sandbox struct {
	GameDifficulty                       int     `ini:"GameDifficulty"`
	LootRespawnEnabled                   bool    `ini:"LootRespawnEnabled"`
	PowerSocketsOffAtNight               bool    `ini:"PowerSocketsOffAtNight"`
	DayNightCycleState                   int     `ini:"DayNightCycleState"`
	DayNightCycleSpeedMultiplier         float64 `ini:"DayNightCycleSpeedMultiplier"`
	SinkRefillRate                       float64 `ini:"SinkRefillRate"`
	FoodSpoilSpeedMultiplier             float64 `ini:"FoodSpoilSpeedMultiplier"`
	RefrigerationEffectivenessMultiplier float64 `ini:"RefrigerationEffectivenessMultiplier"`
	EnemySpawnRate                       float64 `ini:"EnemySpawnRate"`
	EnemyHealthMultiplier                float64 `ini:"EnemyHealthMultiplier"`
	EnemyPlayerDamageMultiplier          float64 `ini:"EnemyPlayerDamageMultiplier"`
	EnemyDeployableDamageMultiplier      float64 `ini:"EnemyDeployableDamageMultiplier"`
	DamageToAlliesMultiplier             float64 `ini:"DamageToAlliesMultiplier"`
	HungerSpeedMultiplier                float64 `ini:"HungerSpeedMultiplier"`
	ThirstSpeedMultiplier                float64 `ini:"ThirstSpeedMultiplier"`
	FatigueSpeedMultiplier               float64 `ini:"FatigueSpeedMultiplier"`
	ContinenceSpeedMultiplier            float64 `ini:"ContinenceSpeedMultiplier"`
	DetectionSpeedMultiplier             float64 `ini:"DetectionSpeedMultiplier"`
	PlayerXPGainMultiplier               float64 `ini:"PlayerXPGainMultiplier"`
	ItemStackSizeMultiplier              float64 `ini:"ItemStackSizeMultiplier"`
	ItemWeightMultiplier                 float64 `ini:"ItemWeightMultiplier"`
	ItemDurabilityMultiplier             float64 `ini:"ItemDurabilityMultiplier"`
	DurabilityLossOnDeathMultiplier      float64 `ini:"ItemDurabilityMultiplier"`
	ShowDeathMessages                    bool    `ini:"ShowDeathMessages"`
	AllowRecipeSharing                   bool    `ini:"AllowRecipeSharing"`
	AllowPagers                          bool    `ini:"AllowPagers"`
	AllowTransmog                        bool    `ini:"AllowTransmog"`
	DisableResearchMinigame              bool    `ini:"DisableResearchMinigame"`
	DeathPenalties                       int     `ini:"DeathPenalties"`
	GlobalRecipeUnlocks                  bool    `ini:"GlobalRecipeUnlocks"`
	FirstTimeStartingWeapon              int     `ini:"FirstTimeStartingWeapon"`
	HostAccessPlayerCorpses              bool    `ini:"HostAccessPlayerCorpses"`
	StorageByTag                         bool    `ini:"StorageByTag"`
	StructuralSupportLimit               int     `ini:"StructuralSupportLimit"`
}
