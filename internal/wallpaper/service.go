package wallpaper

import (
	"os"

	"koboapps/internal/i18n"
)

// Service runs the user-level operations. Sources and ScreensaverDir default to the fixed device
// paths; only tests and the Mac smoke run replace them (the command line cannot).
type Service struct {
	Paths          Paths
	Size           Size
	Sources        []string
	ScreensaverDir string
	Log            *Logger
	// ProgressFile is where a running import or re-fit reports progress ("" = none).
	ProgressFile string
}

// NewService fills the device defaults.
func NewService(paths Paths, size Size, log *Logger) *Service {
	return &Service{Paths: paths, Size: size, Sources: DefaultSources(), ScreensaverDir: ScreensaverFolder(), Log: log}
}

func (service *Service) load() (*Library, Config, error) {
	config, err := LoadConfig(service.Paths.Config())
	if err != nil {
		return nil, Config{}, err
	}
	library, err := LoadLibrary(service.Paths)
	return library, config, err
}

// updateConfig changes one setting on a fresh copy of config.json, so a background job that
// started earlier never overwrites what the user chose meanwhile.
func (service *Service) updateConfig(change func(*Config)) error {
	config, err := LoadConfig(service.Paths.Config())
	if err != nil {
		return err
	}
	change(&config)
	return SaveConfig(service.Paths.Config(), config)
}

// applyCurrentSettings applies with the settings as they are on disk now.
func (service *Service) applyCurrentSettings(library *Library) (ApplyReport, error) {
	config, err := LoadConfig(service.Paths.Config())
	if err != nil {
		return ApplyReport{}, err
	}
	return Apply(library, config, service.ScreensaverDir, service.Log)
}

// importSettled imports with mode, then repeats while the user changed the mode meanwhile, so the
// stored pictures end up fitted for the mode in config.json.
func (service *Service) importSettled(library *Library, mode Mode, kind string) (ImportReport, error) {
	defer clearProgress(service.ProgressFile, service.Log)
	total, err := service.importWithProgress(library, mode, kind)
	for err == nil {
		config, loadErr := LoadConfig(service.Paths.Config())
		if loadErr != nil || config.Mode == mode {
			return total, loadErr
		}
		mode = config.Mode
		var report ImportReport
		report, err = service.importWithProgress(library, mode, ProgressRefit)
		total = total.plus(report)
	}
	return total, err
}

func (service *Service) importWithProgress(library *Library, mode Mode, kind string) (ImportReport, error) {
	library.OnProgress = func(done, total int, name string) {
		progress := Progress{Done: done, Total: total, Name: name, Kind: kind}
		if err := writeProgress(service.ProgressFile, progress); err != nil {
			service.Log.Printf("cannot write progress file %s: %v", service.ProgressFile, err)
		}
	}
	return library.Import(service.Sources, service.Size, mode, service.Log)
}

// ImportAndApply converts new pictures, shows the newest one when rotation is fixed, and applies.
func (service *Service) ImportAndApply() (ImportReport, ApplyReport, error) {
	library, config, err := service.load()
	if err != nil {
		return ImportReport{}, ApplyReport{}, err
	}
	service.ensurePicturesFolder()
	report, err := service.importSettled(library, config.Mode, ProgressImport)
	if err != nil {
		return report, ApplyReport{}, err
	}
	return service.finishImport(library, report)
}

func (service *Service) finishImport(library *Library, report ImportReport) (ImportReport, ApplyReport, error) {
	if report.NewestID != 0 {
		if err := service.updateConfig(func(config *Config) { config.Current = report.NewestID }); err != nil {
			return report, ApplyReport{}, err
		}
	}
	applied, err := service.applyCurrentSettings(library)
	return report, applied, err
}

// ensureWallpapersFolder creates the Pictures folder (the only source) so the first USB copy has a
// target. Failure is only logged: the folder may not exist off-device.
func (service *Service) ensurePicturesFolder() {
	if len(service.Sources) == 0 {
		return
	}
	if err := os.MkdirAll(service.Sources[0], 0o755); err != nil {
		service.Log.Printf("cannot create %s: %v", service.Sources[0], err)
	}
}

// Apply re-applies the current selection.
func (service *Service) Apply() (ApplyReport, error) {
	library, config, err := service.load()
	if err != nil {
		return ApplyReport{}, err
	}
	return Apply(library, config, service.ScreensaverDir, service.Log)
}

// Next selects the following picture and applies it.
func (service *Service) Next() (Entry, error) {
	library, config, err := service.load()
	if err != nil {
		return Entry{}, err
	}
	nextID := library.NextID(config.Current)
	entry, found := library.find(nextID)
	if !found {
		return Entry{}, i18n.NewError("msg.err_no_pictures")
	}
	if err := service.updateConfig(func(config *Config) { config.Current = nextID }); err != nil {
		return Entry{}, err
	}
	_, err = service.applyCurrentSettings(library)
	return entry, err
}

// SaveMode only stores the fit mode. A running background job notices the change when it ends
// and re-fits again; without a job the next import or SetMode applies it.
func (service *Service) SaveMode(mode Mode) error {
	return service.updateConfig(func(config *Config) { config.Mode = mode })
}

// SetMode stores the fit mode at once, re-fits pictures whose originals are still available, and applies.
func (service *Service) SetMode(mode Mode) (ImportReport, ApplyReport, error) {
	if err := service.SaveMode(mode); err != nil {
		return ImportReport{}, ApplyReport{}, err
	}
	library, err := LoadLibrary(service.Paths)
	if err != nil {
		return ImportReport{}, ApplyReport{}, err
	}
	report, err := service.importSettled(library, mode, ProgressRefit)
	if err != nil || len(library.Entries) == 0 {
		return report, ApplyReport{}, err
	}
	applied, err := service.applyCurrentSettings(library)
	return report, applied, err
}

// SetRotation stores fixed or random and applies when there are pictures.
func (service *Service) SetRotation(rotation Rotation) (ApplyReport, error) {
	if err := service.updateConfig(func(config *Config) { config.Rotation = rotation }); err != nil {
		return ApplyReport{}, err
	}
	library, err := LoadLibrary(service.Paths)
	if err != nil || len(library.Entries) == 0 {
		return ApplyReport{}, err
	}
	return service.applyCurrentSettings(library)
}

// TurnOff removes our pictures from the screensaver folder; the library stays.
func (service *Service) TurnOff() (int, error) {
	return Off(service.ScreensaverDir, service.Log)
}

// Status summarizes the state for the status screen.
type Status struct {
	Config   Config
	Pictures int
	Shown    int
}

// Status reads without changing anything.
func (service *Service) Status() (Status, error) {
	library, config, err := service.load()
	if err != nil {
		return Status{}, err
	}
	return Status{Config: config, Pictures: len(library.Entries), Shown: OwnedCount(service.ScreensaverDir)}, nil
}
