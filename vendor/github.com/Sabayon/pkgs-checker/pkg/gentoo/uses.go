/*

Copyright (C) 2017-2021  Daniele Rondina <geaaru@sabayonlinux.org>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.

*/
package gentoo

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type PortageMetaData struct {
	*GentooPackage `json:"package,omitempty"`
	IUse           []string `json:"iuse,omitempty"`
	IUseEffective  []string `json:"iuse_effective,omitempty"`
	Use            []string `json:"use,omitempty"`
	Eapi           string   `json:"eapi,omitempty"`
	CxxFlags       string   `json:"cxxflags,omitempty"`
	CFlags         string   `json:"cflags,omitempty"`
	LdFlags        string   `json:"ldflags,omitempty"`
	CHost          string   `json:"chost,omitempty"`
	BDEPEND        string   `json:"bdepend,omitempty"`
	RDEPEND        string   `json:"rdepend,omitempty"`
	DEPEND         string   `json:"depend,omitempty"`
	REQUIRES       string   `json:"requires,omitempty"`
	KEYWORDS       string   `json:"keywords,omitempty"`
	PROVIDES       string   `json:"provides,omitempty"`
	SIZE           string   `json:"size,omitempty"`
	BUILD_TIME     string   `json:"build_time,omitempty"`
	CBUILD         string   `json:"cbuild,omitempty"`
	COUNTER        string   `json:"counter,omitempty"`
	DEFINED_PHASES string   `json:"defined_phases,omitempty"`
	DESCRIPTION    string   `json:"description,omitempty"`
	FEATURES       string   `json:"features,omitempty"`
	HOMEPAGE       string   `json:"homepage,omitempty"`
	INHERITED      string   `json:"inherited,omitempty"`
	NEEDED         string   `json:"needed,omitempty"`
	NEEDED_ELF2    string   `json:"needed_elf2,omitempty"`
	PKGUSE         string   `json:"pkguse,omitempty"`
	RESTRICT       string   `json:"restrict,omitempty"`

	Ebuild string `json:"ebuild,omitempty"`

	CONTENTS []PortageContentElem `json:"content,omitempty"`
}

type PortageContentElem struct {
	Type          string `json:"type"`
	File          string `json:"file"`
	Hash          string `json:"hash,omitempty"`
	UnixTimestamp string `json:"timestamp,omitempty"`
	Link          string `json:"link,omitempty"`
}

type PortageUseParseOpts struct {
	UseFilters []string `json:"use_filters,omitempty" yaml:"use_filters,omitempty"`
	Categories []string `json:"categories,omitempty" yaml:"categories,omitempty"`
	Packages   []string `json:"pkgs_filters,omitempty" yaml:"pkgs_filters,omitempty"`
	Verbose    bool     `json:"verbose,omitempty" yaml:"verbose,omitempty"`
}

func NewPortageMetaData(pkg *GentooPackage) *PortageMetaData {
	return &PortageMetaData{
		GentooPackage:  pkg,
		IUse:           make([]string, 0),
		IUseEffective:  make([]string, 0),
		Use:            make([]string, 0),
		Eapi:           "",
		CxxFlags:       "",
		LdFlags:        "",
		BDEPEND:        "",
		RDEPEND:        "",
		DEPEND:         "",
		BUILD_TIME:     "",
		CBUILD:         "",
		COUNTER:        "",
		DEFINED_PHASES: "",
		DESCRIPTION:    "",
		FEATURES:       "",
		HOMEPAGE:       "",
		INHERITED:      "",
		NEEDED:         "",
		NEEDED_ELF2:    "",
		PKGUSE:         "",
		RESTRICT:       "",
		REQUIRES:       "",
		SIZE:           "",
		CONTENTS:       make([]PortageContentElem, 0),
	}
}

func (o *PortageUseParseOpts) AddCategory(cat string) {

	present := false
	for _, c := range o.Categories {
		if c == cat {
			present = true
			break
		}
	}

	if !present {
		o.Categories = append(o.Categories, cat)
	}
}

func (o *PortageUseParseOpts) IsCatAdmit(cat string) bool {
	ans := false

	if len(o.Categories) > 0 {
		for _, c := range o.Categories {
			if c == cat {
				ans = true
				break
			}
		}
	} else {
		ans = true
	}

	return ans
}

func (o *PortageUseParseOpts) IsPkgAdmit(pkg string) bool {
	ans := false

	// Prepare regex
	if len(o.Packages) > 0 {

		gp, err := ParsePackageStr(pkg)
		if err == nil && gp.Slot == "0" {
			pkg = gp.GetPackageName() + ":0"
		}

		// Ignoring sub-slot
		if strings.Contains(gp.Slot, "/") {
			gp.Slot = gp.Slot[0:strings.Index(gp.Slot, "/")]
			if o.Verbose {
				fmt.Println(
					fmt.Sprintf("[%s] Force using slot %s", pkg, gp.Slot))
			}
		}

		for _, f := range o.Packages {
			gpF, err := ParsePackageStr(f)
			if err == nil {
				admitted, _ := gpF.Admit(gp)
				if admitted {
					ans = true
					break
				} else if o.Verbose {
					fmt.Println(fmt.Sprintf("[%s] doesn't match %s.",
						pkg, f))
				}
			} else {
				fmt.Println("WARNING: Package " + f + " invalid: " + err.Error() + ".")
			}
		}
	} else {
		ans = true
	}

	return ans
}

func ParseMetadataDir(dir string, opts *PortageUseParseOpts) ([]*PortageMetaData, error) {
	ans := make([]*PortageMetaData, 0)

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return ans, err
	}

	for _, file := range files {
		if file.IsDir() && opts.IsCatAdmit(file.Name()) {
			pkgs, err := ParseMetadataCatDir(filepath.Join(dir, file.Name()), opts)
			if err != nil {
				return ans, errors.New(
					fmt.Sprintf("Error on parse directory %s: %s",
						file.Name(), err.Error()))
			}

			ans = append(ans, pkgs...)
		}
	}

	return ans, nil
}

func ParseMetadataCatDir(dir string, opts *PortageUseParseOpts) ([]*PortageMetaData, error) {
	ans := make([]*PortageMetaData, 0)

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return ans, err
	}

	for _, file := range files {
		if file.IsDir() {
			pm, err := ParsePackageMetadataDir(filepath.Join(dir, file.Name()), opts)
			if err != nil {
				return ans, errors.New(
					fmt.Sprintf("Error on parse directory %s/%s: %s",
						dir, file.Name(), err.Error()))
			}

			if opts.IsPkgAdmit(pm.GetPackageNameWithSlot()) {
				ans = append(ans, pm)
			}
		}
	}

	return ans, nil
}

func ParsePackageMetadataDir(dir string, opts *PortageUseParseOpts) (*PortageMetaData, error) {
	var ans *PortageMetaData = nil

	// Check if the directory is valid
	fi, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}

	if !fi.IsDir() {
		return nil, errors.New("Path " + dir + " is not a directory!")
	}

	metaDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	words := strings.Split(metaDir, "/")

	if len(words) <= 2 {
		return nil, errors.New("Path " + dir + " is invalid!")
	}

	pkgname := fmt.Sprintf("%s/%s",
		words[len(words)-2],
		words[len(words)-1],
	)

	gp, err := ParsePackageStr(pkgname)
	if err != nil {
		return nil, errors.New("Error on parse pkgname " + err.Error())
	}

	ans = NewPortageMetaData(gp)

	ans.BDEPEND, err = parseMetaFile(filepath.Join(metaDir, "BDEPEND"), true)
	if err != nil {
		return nil, err
	}

	ans.RDEPEND, err = parseMetaFile(filepath.Join(metaDir, "RDEPEND"), true)
	if err != nil {
		return nil, err
	}

	ans.DEPEND, err = parseMetaFile(filepath.Join(metaDir, "DEPEND"), true)
	if err != nil {
		return nil, err
	}

	ans.BUILD_TIME, err = parseMetaFile(filepath.Join(metaDir, "BUILD_TIME"), true)
	if err != nil {
		return nil, err
	}

	ans.CBUILD, err = parseMetaFile(filepath.Join(metaDir, "CBUILD"), true)
	if err != nil {
		return nil, err
	}

	ans.COUNTER, err = parseMetaFile(filepath.Join(metaDir, "COUNTER"), true)
	if err != nil {
		return nil, err
	}

	ans.DEFINED_PHASES, err = parseMetaFile(filepath.Join(metaDir, "DEFINED_PHASES"), true)
	if err != nil {
		return nil, err
	}

	ans.DESCRIPTION, err = parseMetaFile(filepath.Join(metaDir, "DESCRIPTION"), true)
	if err != nil {
		return nil, err
	}

	ans.FEATURES, err = parseMetaFile(filepath.Join(metaDir, "FEATURES"), true)
	if err != nil {
		return nil, err
	}

	ans.HOMEPAGE, err = parseMetaFile(filepath.Join(metaDir, "HOMEPAGE"), true)
	if err != nil {
		return nil, err
	}

	ans.INHERITED, err = parseMetaFile(filepath.Join(metaDir, "INHERITED"), true)
	if err != nil {
		return nil, err
	}

	ans.NEEDED, err = parseMetaFile(filepath.Join(metaDir, "NEEDED"), true)
	if err != nil {
		return nil, err
	}

	ans.NEEDED_ELF2, err = parseMetaFile(filepath.Join(metaDir, "NEEDED_ELF2"), true)
	if err != nil {
		return nil, err
	}

	ans.PKGUSE, err = parseMetaFile(filepath.Join(metaDir, "PKGUSE"), true)
	if err != nil {
		return nil, err
	}

	ans.RESTRICT, err = parseMetaFile(filepath.Join(metaDir, "RESTRICT"), true)
	if err != nil {
		return nil, err
	}

	ans.GentooPackage.Slot, err = parseMetaFile(
		filepath.Join(metaDir, "SLOT"), true,
	)
	if err != nil {
		return nil, err
	}

	ans.Eapi, err = parseMetaFile(
		filepath.Join(metaDir, "EAPI"), true,
	)
	if err != nil {
		return nil, err
	}

	ans.CFlags, err = parseMetaFile(
		filepath.Join(metaDir, "CFLAGS"), true,
	)
	if err != nil {
		return nil, err
	}

	ans.CxxFlags, err = parseMetaFile(
		filepath.Join(metaDir, "CXXFLAGS"), true,
	)
	if err != nil {
		return nil, err
	}

	ans.LdFlags, err = parseMetaFile(
		filepath.Join(metaDir, "LDFLAGS"), true,
	)
	if err != nil {
		return nil, err
	}

	ans.CHost, err = parseMetaFile(
		filepath.Join(metaDir, "CHOST"), true,
	)
	if err != nil {
		return nil, err
	}

	ans.GentooPackage.License, err = parseMetaFile(
		filepath.Join(metaDir, "LICENSE"), true,
	)
	if err != nil {
		return nil, err
	}

	ans.GentooPackage.Repository, err = parseMetaFile(
		filepath.Join(metaDir, "repository"), true,
	)
	if err != nil {
		return nil, err
	}

	ans.REQUIRES, err = parseMetaFile(
		filepath.Join(metaDir, "REQUIRES"), true,
	)
	if err != nil {
		return nil, err
	}

	ans.KEYWORDS, err = parseMetaFile(
		filepath.Join(metaDir, "KEYWORDS"), true,
	)
	if err != nil {
		return nil, err
	}

	ans.PROVIDES, err = parseMetaFile(
		filepath.Join(metaDir, "PROVIDES"), true,
	)
	if err != nil {
		return nil, err
	}

	ans.SIZE, err = parseMetaFile(
		filepath.Join(metaDir, "SIZE"), true,
	)
	if err != nil {
		return nil, err
	}

	iuse, err := parseMetaFile(
		filepath.Join(metaDir, "IUSE"), true,
	)
	if err != nil {
		return nil, err
	}
	if iuse != "" {
		ans.IUse = strings.Split(iuse, " ")
	}

	iuseEffective, err := parseMetaFile(
		filepath.Join(metaDir, "IUSE_EFFECTIVE"), true,
	)
	if err != nil {
		return nil, err
	}
	if iuseEffective != "" {
		ans.IUseEffective = strings.Split(iuseEffective, " ")
	}

	use, err := parseMetaFile(
		filepath.Join(metaDir, "USE"), true,
	)
	if err != nil {
		return nil, err
	}
	if use != "" {
		ans.Use = strings.Split(use, " ")
	}

	if len(ans.IUseEffective) > 0 {
		ans.GentooPackage.UseFlags = elaborateUses(ans.IUseEffective, ans.Use, opts)
	}

	ans.Ebuild, err = parseMetaFile(filepath.Join(metaDir, ans.GentooPackage.GetPF()+".ebuild"), true)
	if err != nil {
		return nil, err
	}

	ans.CONTENTS, err = GetCONTENTS(filepath.Join(metaDir, "CONTENTS"))
	if err != nil {
		return nil, err
	}

	return ans, nil
}

func useInArray(use string, arr []string) bool {
	ans := false
	for _, u := range arr {
		if use == u {
			ans = true
			break
		}
	}
	return ans
}

func elaborateUses(iuse, use []string, opts *PortageUseParseOpts) []string {
	ans := []string{}

	// Prepare regex
	listRegex := []*regexp.Regexp{}
	for _, f := range opts.UseFilters {
		r := regexp.MustCompile(f)
		if r != nil {
			listRegex = append(listRegex, r)
		} else {
			fmt.Println("WARNING: Regex " + f + " not compiled.")
		}
	}

	for _, u := range iuse {

		toSkip := false

		if strings.HasPrefix(u, "+") {
			u = u[1:]
		}

		// Check if use flags is filtered
		if len(listRegex) > 0 {
			for _, r := range listRegex {
				if r.MatchString(u) {
					toSkip = true
					//fmt.Println("MATCHED FILTER ", u)
					break
				}
			}
		}

		if toSkip {
			continue
		}

		if useInArray(u, use) {
			ans = append(ans, u)
		} else {
			ans = append(ans, "-"+u)
		}
	}

	return ans
}

func parseMetaFile(file string, dropLn bool) (string, error) {
	_, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}

	ans := string(data)
	if dropLn {
		ans = strings.TrimRight(ans, "\n")
	}

	return ans, nil
}

func GetCONTENTS(file string) ([]PortageContentElem, error) {
	ans := []PortageContentElem{}

	_, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return ans, nil
		}
		return ans, err
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return ans, err
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {

		words := strings.Split(line, " ")

		if len(words) > 1 {

			// File with spaces are available in multiple words.

			elem := PortageContentElem{
				Type: words[0],
			}

			switch words[0] {
			case "dir":
				elem.File = strings.Join(words[1:], " ")
			case "obj":
				elem.File = strings.Join(words[1:len(words)-2], " ")
				elem.Hash = strings.Join(words[len(words)-2:len(words)-1], " ")
				elem.UnixTimestamp = words[len(words)-1]
			case "sym":
				elem.File = strings.Join(words[1:len(words)-3], " ")
				elem.Link = words[len(words)-2]
				elem.UnixTimestamp = words[len(words)-1]
			default:
				continue
			}

			ans = append(ans, elem)
		}

	}

	return ans, nil
}

func (e PortageContentElem) String() string {
	ans := e.Type + " "
	ans += e.File

	switch e.Type {
	case "obj":
		ans += " " + e.Hash + " " + e.UnixTimestamp
	case "sym":
		ans += " " + "-> " + e.Link + " " + e.UnixTimestamp
	default:
	}
	return ans
}

func (m *PortageMetaData) WriteMetadata2Dir(dir string, opts *PortageUseParseOpts) error {

	metadir := filepath.Join(dir,
		fmt.Sprintf("/var/db/pkg/%s-%s", m.GetPackageName(), m.GetPVR()),
	)

	var fileMode os.FileMode
	fileMode = os.ModeDir | 0744

	err := os.MkdirAll(metadir, fileMode)
	if err != nil {
		return err
	}

	// Write BDEPEND file
	if m.BDEPEND != "" {
		err = os.WriteFile(filepath.Join(metadir, "BDEPEND"),
			[]byte(m.BDEPEND+"\n"), 0644,
		)
		if err != nil {
			return err
		}
	}

	// Write BUILD_TIME
	err = os.WriteFile(filepath.Join(metadir, "BUILD_TIME"),
		[]byte(m.BUILD_TIME+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write CATEGORY
	err = os.WriteFile(filepath.Join(metadir, "CATEGORY"),
		[]byte(m.Category+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write CBUILD
	err = os.WriteFile(filepath.Join(metadir, "CBUILD"),
		[]byte(m.CBUILD+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write CFLAGS
	err = os.WriteFile(filepath.Join(metadir, "CFLAGS"),
		[]byte(m.CFlags+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write CHOST
	err = os.WriteFile(filepath.Join(metadir, "CHOST"),
		[]byte(m.CHost+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write CONTENTS
	if len(m.CONTENTS) > 0 {
		contents := ""
		for _, e := range m.CONTENTS {
			contents += e.String() + "\n"
		}
		// TODO: maybe this could be handled with a writer that
		// doesn't require the load of all files in memory.
		err = os.WriteFile(filepath.Join(metadir, "CONTENTS"),
			[]byte(contents), 0644,
		)

		if err != nil {
			return errors.New("Error on write CONTENTS: " + err.Error())
		}
	}

	// Write COUNTER
	err = os.WriteFile(filepath.Join(metadir, "COUNTER"),
		[]byte(m.COUNTER), 0644,
	)
	if err != nil {
		return err
	}

	// Write CXXFLAGS
	err = os.WriteFile(filepath.Join(metadir, "CXXFLAGS"),
		[]byte(m.CxxFlags+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write DEFINED_PHASES
	err = os.WriteFile(filepath.Join(metadir, "DEFINED_PHASES"),
		[]byte(m.DEFINED_PHASES+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write DEPEND
	if m.DEPEND != "" {
		err = os.WriteFile(filepath.Join(metadir, "DEPEND"),
			[]byte(m.DEPEND+"\n"), 0644,
		)
		if err != nil {
			return err
		}
	}

	// Write DESCRIPTION
	err = os.WriteFile(filepath.Join(metadir, "DESCRIPTION"),
		[]byte(m.DESCRIPTION+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write EAPI
	err = os.WriteFile(filepath.Join(metadir, "EAPI"),
		[]byte(m.Eapi+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write FEATURES
	err = os.WriteFile(filepath.Join(metadir, "FEATURES"),
		[]byte(m.FEATURES+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write HOMEPAGE
	if m.HOMEPAGE != "" {
		err = os.WriteFile(filepath.Join(metadir, "HOMEPAGE"),
			[]byte(m.HOMEPAGE+"\n"), 0644,
		)
		if err != nil {
			return err
		}
	}

	// Write INHERITED
	if m.INHERITED != "" {
		err = os.WriteFile(filepath.Join(metadir, "INHERITED"),
			[]byte(m.INHERITED+"\n"), 0644,
		)
		if err != nil {
			return err
		}
	}

	// Write IUSE
	err = os.WriteFile(filepath.Join(metadir, "IUSE"),
		[]byte(strings.Join(m.IUse, " ")+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write IUSE_EFFECTIVE
	err = os.WriteFile(filepath.Join(metadir, "IUSE_EFFECTIVE"),
		[]byte(strings.Join(m.IUseEffective, " ")+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write KEYWORDS
	err = os.WriteFile(filepath.Join(metadir, "KEYWORDS"),
		[]byte(m.KEYWORDS+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write LDFLAGS
	err = os.WriteFile(filepath.Join(metadir, "LDFLAGS"),
		[]byte(m.LdFlags+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write LICENSE
	err = os.WriteFile(filepath.Join(metadir, "LICENSE"),
		[]byte(m.GentooPackage.License+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write NEEDED
	if m.NEEDED != "" {
		err = os.WriteFile(filepath.Join(metadir, "NEEDED"),
			[]byte(m.NEEDED+"\n"), 0644,
		)
		if err != nil {
			return err
		}
	}

	// Write NEEDED.ELF.2
	if m.NEEDED_ELF2 != "" {
		err = os.WriteFile(filepath.Join(metadir, "NEEDED.ELF.2"),
			[]byte(m.NEEDED_ELF2+"\n"), 0644,
		)
		if err != nil {
			return err
		}
	}

	// Write PF
	err = os.WriteFile(filepath.Join(metadir, "PF"),
		[]byte(m.GetPF()+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write PKGUSE
	if m.PKGUSE != "" {
		err = os.WriteFile(filepath.Join(metadir, "PKGUSE"),
			[]byte(m.PKGUSE+"\n"), 0644,
		)
		if err != nil {
			return err
		}
	}

	// Write PROVIDES
	if m.PROVIDES != "" {
		err = os.WriteFile(filepath.Join(metadir, "PROVIDES"),
			[]byte(m.PROVIDES+"\n"), 0644,
		)
		if err != nil {
			return err
		}
	}

	// Write RDEPEND
	if m.RDEPEND != "" {
		err = os.WriteFile(filepath.Join(metadir, "RDEPEND"),
			[]byte(m.RDEPEND+"\n"), 0644,
		)
		if err != nil {
			return err
		}
	}

	// Write repository
	err = os.WriteFile(filepath.Join(metadir, "repository"),
		[]byte(m.GentooPackage.Repository+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write REQUIRES
	if m.REQUIRES != "" {
		err = os.WriteFile(filepath.Join(metadir, "REQUIRES"),
			[]byte(m.REQUIRES+"\n"), 0644,
		)
		if err != nil {
			return err
		}
	}

	// Write RESTRICT
	if m.RESTRICT != "" {
		err = os.WriteFile(filepath.Join(metadir, "RESTRICT"),
			[]byte(m.RESTRICT+"\n"), 0644,
		)
		if err != nil {
			return err
		}
	}

	// Write SIZE
	err = os.WriteFile(filepath.Join(metadir, "SIZE"),
		[]byte(m.SIZE+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(metadir, "SLOT"),
		[]byte(m.GentooPackage.Slot+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write USE
	err = os.WriteFile(filepath.Join(metadir, "USE"),
		[]byte(strings.Join(m.Use, " ")+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	// Write <name>-<version>.ebuild
	err = os.WriteFile(filepath.Join(metadir,
		fmt.Sprintf("%s.ebuild", m.GetPF())),
		[]byte(m.Ebuild+"\n"), 0644,
	)
	if err != nil {
		return err
	}

	return nil
}
