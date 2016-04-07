package ps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/jweir/csv"
	"gopkg.in/validator.v2"
	"regexp"

	"path/filepath"
	"strings"
)

type PrestaShop struct {
	Config     Config
	Products   []Product
	Categories []Category
	Cookies    []*http.Cookie
	token      string

	tmpdir string

	//represent the csv files that need to be uploaded
	products2upload []string
	//represent the csv files previously uploaded and renamed by prestashop that need to be imported
	products2import []string

	//represent the csv files that need to be uploaded
	categories2upload []string
	//represent the csv files previously uploaded and renamed by prestashop  that need to be imported
	categories2import []string
}

//Creates a new instance of Prestashop Importer
func New(c Config) *PrestaShop {
	return &PrestaShop{Config: c}
}

//Init Tries to Login to the Prestashop store
func (ps *PrestaShop) Init() error {
	var err error
	ps.tmpdir, err = ioutil.TempDir("", "prestashop")
	if err != nil {
		return err
	}
	return ps.auth()
}

func (ps *PrestaShop) auth() (err error) {

	data := url.Values{}
	//"email" => "YOUR_ADMIN_EMAIL","passwd" => "YOUR_ADMIN_PASS", "submitLogin" => "Connexion"
	data.Add("email", ps.Config.Email)
	data.Add("passwd", ps.Config.Password)
	data.Add("submitLogin", "1")
	data.Add("token", "")
	data.Add("ajax", "1")
	data.Add("controller", "AdminLogin")
	data.Add("redirect", "&token="+ps.token)
	resp, err := http.PostForm(ps.Config.AdminUrl+"ajax-tab.php?rand=1435224829042", data)
	if err != nil {
		return
	}
	if ps.Config.Debug {
		log.Println(resp.Cookies())
	}

	var authResp AuthResp

	buffer := &bytes.Buffer{}

	_, err = buffer.ReadFrom(resp.Body)
	if err != nil {
		return
	}

	err = json.NewDecoder(buffer).Decode(&authResp)
	if err != nil {
		return
	}

	ps.Cookies = resp.Cookies()
	return
}

func (p *PrestaShop) SetProducts(products []Product) error {
	p.Products = products
	return p.validateProducts()
}

func (p *PrestaShop) SetCategories(products []Product) error {
	p.Products = products
	return p.validateCategories()
}

func (p *PrestaShop) validateProducts() error {

	for i := range p.Products {
		if err := validator.Validate(p.Products[i]); err != nil {
			return err
		}
	}

	return nil
}

func (p *PrestaShop) validateCategories() error {
	for i := range p.Categories {
		if err := validator.Validate(p.Categories[i]); err != nil {
			return err
		}
	}
	return nil
}

func (ps *PrestaShop) ImportProducts() error {

	err := ps.genProductCSVs()
	if err != nil {
		return err
	}

	err = ps.auth()
	if err != nil {
		return err
	}

	for i := range ps.products2upload {
		err = ps.uploadCSV(ps.products2upload[i])
		if err != nil {
			return err
		}
	}

	for i := range ps.products2import {
		err = ps.importCSVproducts(ps.products2import[i])
		if err != nil {
			return err
		}
		if !ps.Config.Debug {
			os.Remove(filepath.Join(ps.tmpdir, ps.products2import[i]))
		}
	}

	return nil
}
func (ps *PrestaShop) ImportCategories() error {
	err := ps.genCategoriesCSVs()
	if err != nil {
		return err
	}

	err = ps.auth()
	if err != nil {
		return err
	}

	for i := range ps.categories2upload {
		err = ps.uploadCSV(ps.products2upload[i])
		if err != nil {
			return err
		}
	}

	for i := range ps.categories2import {
		err = ps.importCSVcategories(ps.categories2import[i])
		if err != nil {
			return err
		}
		if !ps.Config.Debug {
			os.Remove(filepath.Join(ps.tmpdir, ps.categories2import[i]))
		}
	}

	return nil
}

func (ps *PrestaShop) genProductCSVs() error {

	var products []Product
	var nfiles int

	for i := range ps.Products {

		if ps.Config.SkipFirstRecord && i == 0 {
			continue
		}

		products = append(products, ps.Products[i])

		if i%ps.Config.MaxItemsPerFile == 0 {
			data, err := csv.Marshal(products)
			if err != nil {
				return err
			}

			csvfile := filepath.Join(ps.tmpdir, "out-"+strconv.Itoa(nfiles)+".csv")

			err = ioutil.WriteFile(csvfile, data, 0777)
			if err != nil {
				return err
			}

			ps.products2upload = append(ps.products2upload, csvfile)

			nfiles++
			products = []Product{}
		}

		if i == len(ps.Products)-1 {
			data, err := csv.Marshal(products)
			if err != nil {
				return err
			}
			csvfile := filepath.Join(ps.tmpdir, "out-"+strconv.Itoa(nfiles)+".csv")

			err = ioutil.WriteFile(csvfile, data, 0777)
			if err != nil {
				return err
			}

			ps.products2upload = append(ps.products2upload, csvfile)
		}
	}

	return nil
}

func (ps *PrestaShop) genCategoriesCSVs() error {

	var categories []Category
	var nfiles int

	for i := range ps.Categories {

		if ps.Config.SkipFirstRecord && i == 0 {
			continue
		}

		categories = append(categories, ps.Categories[i])

		if i%ps.Config.MaxItemsPerFile == 0 {
			data, err := csv.Marshal(categories)
			if err != nil {
				return err
			}

			csvfile := filepath.Join(ps.tmpdir, "out-cat-"+strconv.Itoa(nfiles)+".csv")

			err = ioutil.WriteFile(csvfile, data, 0777)
			if err != nil {
				return err
			}

			ps.categories2upload = append(ps.categories2upload, csvfile)

			nfiles++
			categories = []Category{}
		}

		if i == len(ps.Categories)-1 {
			data, err := csv.Marshal(categories)
			if err != nil {
				return err
			}
			csvfile := filepath.Join(ps.tmpdir, "out-cat-"+strconv.Itoa(nfiles)+".csv")

			err = ioutil.WriteFile(csvfile, data, 0777)
			if err != nil {
				return err
			}

			ps.categories2upload = append(ps.categories2upload, csvfile)
		}
	}

	return nil
}

func (ps *PrestaShop) uploadCSV(filename string) (err error) {
	// Prepare a form that you will submit to that URL.
	var tries = 3
again:
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	// Add your image file
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return
	}
	if _, err = io.Copy(fw, f); err != nil {
		return
	}

	data := url.Values{}
	data.Add("token", ps.token)
	data.Add("ajax", "1")
	data.Add("action", "uploadCsv")
	data.Add("rand", "1435224829042")
	data.Add("controller", "AdminImport")

	// Add the other fields
	if fw, err = w.CreateFormField("multiple_value_separator"); err != nil {
		return
	}
	if _, err = fw.Write([]byte(",")); err != nil {
		return
	}
	if fw, err = w.CreateFormField("separator"); err != nil {
		return
	}
	if _, err = fw.Write([]byte(";")); err != nil {
		return
	}
	if fw, err = w.CreateFormField("iso_lang"); err != nil {
		return
	}
	if _, err = fw.Write([]byte("pt")); err != nil {
		return
	}
	if fw, err = w.CreateFormField("entity"); err != nil {
		return
	}
	if _, err = fw.Write([]byte("0")); err != nil {
		return
	}
	if fw, err = w.CreateFormField("forceIDs"); err != nil {
		return
	}
	if _, err = fw.Write([]byte("on")); err != nil {
		return
	}
	if fw, err = w.CreateFormField("regenerate"); err != nil {
		return
	}
	if _, err = fw.Write([]byte("on")); err != nil {
		return
	}
	if fw, err = w.CreateFormField("csv"); err != nil {
		return
	}
	if _, err = fw.Write([]byte(filename)); err != nil {
		return
	}

	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", ps.Config.AdminUrl+"?"+data.Encode(), &b)
	if err != nil {
		return
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType()+"; charset=utf-8")

	for i := range ps.Cookies {
		req.AddCookie(ps.Cookies[i])
	}

	// Submit the request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return
	}

	// Check the response
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", res.Status)
	}

	var buf = &bytes.Buffer{}

	_, err = buf.ReadFrom(res.Body)
	if err != nil {
		return
	}

	var uploadResp UploadResp
	err = json.NewDecoder(buf).Decode(&uploadResp)
	if err != nil {
		if tries > 0 {
			tries--
			if bytes.Contains(buf.Bytes(), []byte("Token de segurança inválido")) {

				ps.getToken(buf.String())
				ps.auth()
				goto again
			} else {
				return

			}
		} else {
			return
		}

	}

	if ps.Config.Debug {
		log.Println(uploadResp)
	}
	ps.products2import = append(ps.products2import, uploadResp.File.Filename)
	return
}

func (ps *PrestaShop) importCSVproducts(filename string) error {
	data := url.Values{}
	data.Add("csv", filename)
	//data.Add("convert", "on")
	//data.Add("regenerate", "on")
	data.Add("entity", "1")
	data.Add("iso_lang", "pt")
	data.Add("match_ref", "1")
	//data.Add("forceIDs", "1")
	data.Add("separator", ",")
	data.Add("multiple_value_separator", "|")
	data.Add("skip", "1")
	data.Add("type_value[0]", "id")
	data.Add("type_value[1]", "active")
	data.Add("type_value[2]", "name")
	data.Add("type_value[3]", "category")
	data.Add("type_value[4]", "price_tex")
	data.Add("type_value[5]", "id_tax_rules_group")
	data.Add("type_value[6]", "wholesale_price")
	data.Add("type_value[7]", "on_sale")
	data.Add("type_value[8]", "reduction_price")
	data.Add("type_value[9]", "reduction_percent")
	data.Add("type_value[10]", "reduction_from")
	data.Add("type_value[11]", "reduction_to")
	data.Add("type_value[12]", "reference")
	data.Add("type_value[13]", "supplier_reference")
	data.Add("type_value[14]", "supplier")
	data.Add("type_value[15]", "manufacturer")
	data.Add("type_value[16]", "ean13")
	data.Add("type_value[17]", "upc")
	data.Add("type_value[18]", "ecotax")
	data.Add("type_value[19]", "width")
	data.Add("type_value[20]", "height")
	data.Add("type_value[21]", "depth")
	data.Add("type_value[22]", "weight")
	data.Add("type_value[23]", "quantity")
	data.Add("type_value[24]", "minimal_quantity")
	data.Add("type_value[25]", "visibility")
	data.Add("type_value[26]", "additional_shipping_cost")
	data.Add("type_value[27]", "unity")
	data.Add("type_value[28]", "unit_price")
	data.Add("type_value[29]", "description_short")
	data.Add("type_value[30]", "description")
	data.Add("type_value[31]", "tags")
	data.Add("type_value[32]", "meta_title")
	data.Add("type_value[33]", "meta_keywords")
	data.Add("type_value[34]", "meta_description")
	data.Add("type_value[35]", "link_rewrite")
	data.Add("type_value[36]", "available_now")
	data.Add("type_value[37]", "available_later")
	data.Add("type_value[38]", "available_for_order")
	data.Add("type_value[39]", "available_date")
	data.Add("type_value[40]", "date_add")
	data.Add("type_value[41]", "show_price")
	data.Add("type_value[42]", "image")
	data.Add("type_value[43]", "delete_existing_images")
	data.Add("type_value[44]", "features")
	data.Add("type_value[45]", "online_only")
	data.Add("type_value[46]", "condition")
	data.Add("type_value[47]", "customizable")
	data.Add("type_value[48]", "uploadable_files")
	data.Add("type_value[49]", "text_fields")
	data.Add("type_value[50]", "out_of_stock")
	data.Add("type_value[51]", "shop")
	data.Add("type_value[52]", "advanced_stock_management")
	data.Add("type_value[53]", "depends_on_stock")
	data.Add("type_value[54]", "warehouse")
	data.Add("import", "")

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", ps.Config.AdminUrl+"?"+"controller=AdminImport&token="+ps.token, bytes.NewReader([]byte(data.Encode())))
	if err != nil {
		return err
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	for i := range ps.Cookies {
		req.AddCookie(ps.Cookies[i])
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer([]byte(""))

	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return err
	}

	// Check the response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s  %s  ", resp.Status, buf.String())
	}

	if ps.Config.Debug {
		log.Println(filename, "Imported", resp.Status)
	}

	return nil
}

func (ps *PrestaShop) importCSVcategories(filename string) error {
	data := url.Values{}
	data.Add("csv", filename)
	data.Add("convert", "")
	data.Add("regenerate", "on")
	data.Add("entity", "0")
	data.Add("iso_lang", "pt")
	data.Add("separator", ";")
	data.Add("multiple_value_separator", ",")
	data.Add("skip", "1")
	data.Add("type_value[0]", "id")
	data.Add("type_value[1]", "active")
	data.Add("type_value[2]", "name")
	data.Add("type_value[3]", "parent")
	data.Add("type_value[4]", "is_root_category")
	data.Add("type_value[5]", "description")
	data.Add("type_value[6]", "meta_title")
	data.Add("type_value[7]", "meta_keywords")
	data.Add("type_value[8]", "meta_description")
	data.Add("import", "")

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", ps.Config.AdminUrl+"?"+"controller=AdminImport&token="+ps.token, bytes.NewReader([]byte(data.Encode())))
	if err != nil {
		return err
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	for i := range ps.Cookies {
		req.AddCookie(ps.Cookies[i])
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer([]byte(""))

	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return err
	}

	// Check the response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	if ps.Config.Debug {
		log.Println(filename, "Imported", resp.Status)
	}

	//log.Println(buf.String())
	return nil
}

func (ps *PrestaShop) getToken(buf string) error {
	regex := regexp.MustCompile("token=([a-z0-9]+)")
	ps.token = strings.TrimPrefix(regex.FindString(buf), "token=")
	return nil
}
