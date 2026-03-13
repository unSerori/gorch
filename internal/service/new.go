package service

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pelletier/go-toml/v2"
	"github.com/unSerori/gorch/internal/model"
)

type NewService struct {
	// ここにinfraの依存
}

type Preset struct {
	Name        string     `toml:"name"`
	Version     string     `toml:"version"`
	Description string     `toml:"description"`
	Files       []FileSpec `toml:"files"`
}

type FileSpec struct {
	Tmpl string `toml:"tmpl"`
	Out  string `toml:"out"`
}

func (s *NewService) CreatePJ(presetName string) error { // ここでCmdからの引数を受けとる。たぶんプリセット名
	fmt.Printf("here is CreatePJ in NewService. arg1 is %s\n", presetName)

	presetPath, err := s.resolvePresetDir(presetName)
	if err != nil {
		return err
	}

	preset, err := s.loadPresetConfig(presetPath, presetName)
	if err != nil {
		return err
	}

	fmt.Printf("Preset.Name: %s\n", preset.Name)
	fmt.Printf("Preset.Version: %s\n", preset.Version)
	fmt.Printf("Preset.Description: %s\n", preset.Description)
	fmt.Printf("Preset.Files: %s\n", preset.Files)
	fmt.Println()
	fmt.Printf("Preset.Files[0].Tmpl: %s\n", preset.Files[0].Tmpl)
	fmt.Printf("Preset.Files[0].Out: %s\n", preset.Files[0].Out)
	fmt.Printf("Preset.Files[1].Tmpl: %s\n", preset.Files[1].Tmpl)
	fmt.Printf("Preset.Files[1].Out: %s\n", preset.Files[1].Out)
	fmt.Printf("Preset.Files[2].Tmpl: %s\n", preset.Files[2].Tmpl)
	fmt.Printf("Preset.Files[2].Out: %s\n", preset.Files[2].Out)

	values, err := s.loadValues(presetPath)
	if err != nil {
		return err
	}

	err = s.generateFiles(presetPath, preset.Files, values)
	if err != nil {
		return err
	}

	return nil
}

// プリセットを探す。
// HACK: 検索順序。特定ディレクトリ->組み込みプリセット
// HACK: プリセットの中身が異常ではない（仕様通り）か。
// HACK: 探す処理切り出した方がいいかも？
func (s *NewService) resolvePresetDir(presetName string) (model.Path, error) {
	// HACK: presets/は一旦ハードコードで。あとあとデフォルト値に対してViperによる指定上書きを実装
	presetsDir := "presets"
	presetDir := filepath.Join(presetsDir, presetName)

	_, err := os.Stat(presetDir)
	if err != nil { // 存在確認だけでなく権限や読み込みエラーなども確認したいため、if os.IsNotExistを使っていない。
		return "", err
	}

	return model.Path(presetDir), nil
}

// i/oとTomlパーサでプリセットの設定を読み込む
func (s *NewService) loadPresetConfig(presetDir model.Path, presetName string) (*Preset, error) {
	bytes, err := os.ReadFile(filepath.Join(string(presetDir), "preset.toml"))
	if err != nil {
		return nil, err
	}

	var preset Preset
	err = toml.Unmarshal(bytes, &preset)
	if err != nil {
		return nil, err
	}

	// HACK: し、念の為プリセット名を確認、警告を出す。
	if preset.Name != presetName {
		fmt.Println("") // HACK: logger // HACK: i18n
	}

	return &preset, nil
}

// 流し込む値を取得
func (s *NewService) loadValues(presetDir model.Path) (map[string]any, error) {
	bytes, err := os.ReadFile(filepath.Join(string(presetDir), "default.toml"))
	if err != nil {
		return nil, err
	}

	var values map[string]any
	err = toml.Unmarshal(bytes, &values)
	if err != nil {
		return nil, err
	}

	return values, nil
}

// []FileSpecを元にファイル群を生成する
func (s *NewService) generateFiles(presetDir model.Path, files []FileSpec, values map[string]any) error {
	// すべてのFileSpecに対して。
	for _, f := range files {
		fulTmpl := filepath.Join(string(presetDir), "tmpls", f.Tmpl)

		renderedTmpl, err := s.renderTmpl(model.Path(fulTmpl), values)
		if err != nil {
			return err
		}

		// HACK: オプションでのディレクトリ指定を行う場合、HACK: ここで指定された場所とmodel.Path(f.Out)を結合する。
		dest := model.Path(f.Out)

		err = s.atomicWriteFile(dest, renderedTmpl)
		if err != nil {
			return err
		}
	}

	return nil
}

// text/templateでテンプレートを組み立てる。
func (s *NewService) renderTmpl(tmplPath model.Path, values map[string]any) ([]byte, error) {
	tmpl, err := template.ParseFiles(string(tmplPath))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, values)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
	// return []byte("aa"), nil
}

// ファイルを生成
// HACK: atomicにする。
func (s *NewService) atomicWriteFile(dest model.Path, content []byte) error {
	// HACK: パーミッションも設定ファイルの値から決定する。そのため一旦変数で宣言
	dirPerm := os.FileMode(0o755)
	filePerm := os.FileMode(0o644)

	dir := filepath.Dir(string(dest))
	err := os.MkdirAll(dir, dirPerm) // ここってすでに存在したらどうなるのか？
	if err != nil {
		return err
	}

	// 存在しないなら作成。
	// NOTE: 存在した場合に上書きしないために、WriteFileやCreateを使っていない。
	file, err := os.OpenFile(string(dest), os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePerm) // 書き込み、新規、存在するなら禁止
	if err != nil {
		if os.IsExist(err) {
			// // HACK: 強制オプションなど？するならここの条件でさらにオプション確認
			// if 強制オプションの判定 {
			//  if err := os.WriteFile(string(dest), content, filePerm); err != nil {
			//      return err
			//  }
			//  fmt.Println("強制オプションによって上書き") // HACK: logger // HACK: i18n

			// } else {
			//  // HACK: 警告出してスルー
			//  fmt.Println("そんざいするからスキップするよというログ") // HACK: logger // HACK: i18n
			//  return nil
			// }
			// HACK: 警告出してスルー
			fmt.Println("そんざいするからスキップするよというログ") // HACK: logger // HACK: i18n
			return nil
		}
		return err
	}
	defer file.Close()
	_, err = file.Write(content)
	if err != nil {
		return err
	}

	return nil
}
