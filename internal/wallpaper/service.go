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

// ImportAndApply converts new pictures, shows the newest one when rotation is fixed, and applies.
func (service *Service) ImportAndApply() (ImportReport, ApplyReport, error) {
	library, config, err := service.load()
	if err != nil {
		return ImportReport{}, ApplyReport{}, err
	}
	service.ensurePicturesFolder()
	report, err := library.Import(service.Sources, service.Size, config.Mode, service.Log)
	if err != nil {
		return report, ApplyReport{}, err
	}
	if report.NewestID != 0 {
		config.Current = report.NewestID
	}
	return service.saveAndApply(library, config, report)
}

func (service *Service) saveAndApply(library *Library, config Config, report ImportReport) (ImportReport, ApplyReport, error) {
	if err := SaveConfig(service.Paths.Config(), config); err != nil {
		return report, ApplyReport{}, err
	}
	applied, err := Apply(library, config, service.ScreensaverDir, service.Log)
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
	config.Current = nextID
	_, _, err = service.saveAndApply(library, config, ImportReport{})
	return entry, err
}

// SetMode stores the fit mode, re-fits pictures whose originals are still available, and applies.
func (service *Service) SetMode(mode Mode) (ImportReport, ApplyReport, error) {
	library, config, err := service.load()
	if err != nil {
		return ImportReport{}, ApplyReport{}, err
	}
	config.Mode = mode
	report, err := library.Import(service.Sources, service.Size, mode, service.Log)
	if err != nil {
		return report, ApplyReport{}, err
	}
	if len(library.Entries) == 0 {
		return report, ApplyReport{}, SaveConfig(service.Paths.Config(), config)
	}
	return service.saveAndApply(library, config, report)
}

// SetRotation stores fixed or random and applies when there are pictures.
func (service *Service) SetRotation(rotation Rotation) (ApplyReport, error) {
	library, config, err := service.load()
	if err != nil {
		return ApplyReport{}, err
	}
	config.Rotation = rotation
	if len(library.Entries) == 0 {
		return ApplyReport{}, SaveConfig(service.Paths.Config(), config)
	}
	_, applied, err := service.saveAndApply(library, config, ImportReport{})
	return applied, err
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
