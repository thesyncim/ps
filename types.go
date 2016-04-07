package ps

//Product represent a PrestaShop Product
type Product struct {
	ID                       string `validate:"nonzero"`
	Active                   string
	Name                     string `validate:"nonzero"`
	Categories               string
	Price                    float64
	TaxrulesID               string
	Wholesaleprice           string
	Onsale                   string // (0/1)
	Discountamount           string
	Discountpercent          string
	Discountfromb            string // (yyyy-mm-dd)
	Discountto               string //(yyyy-mm-dd)
	Reference                string //
	Supplierreference        string //#
	Supplier                 string //
	Manufacturer             string //
	EAN13                    string //
	UPC                      string //
	Ecotax                   string
	Width                    string
	Height                   string
	Depth                    string
	Weight                   string
	Quantity                 string
	Minimalquantity          string
	Visibility               string
	Additionalshippingcost   string
	Unity                    string
	Unitprice                string
	Shortdescription         string
	Description              string `validate:"nonzero"`
	Tags                     string //(x,y,z...)
	Metatitle                string
	Metakeywords             string
	Metadescription          string
	URLrewritten             string
	Textwheninstock          string
	Textwhenbackorderallowed string
	Availablefororder        string // (0 = No, 1 = Yes)
	Productavailabledate     string
	Productcreationdate      string
	Showprice                string //(0 = No, 1 = Yes)
	ImageURLs                string //(x,y,z...)
	Deleteexistingimages     string //(0 = No, 1 = Yes)
	Feature                  string //(Name:Value:Position)
	Availableonlineonly      string // (0 = No, 1 = Yes)
	Condition                string
	Customizable             string // (0 = No, 1 = Yes)
	Uploadablefiles          string //(0 = No, 1 = Yes)
	Textfields               string // (0 = No, 1 = Yes)
	Outofstock               string
	IDShop                   string // Name of shop
	Advancedstockmanagement  string
	DependsOnStock           string
	Warehouse                string
}

type Category struct {
	ID             string `validate:"nonzero"`
	Active         string
	Name           string `validate:"nonzero"`
	ParentCategory string
	Rootcategory   string //(0/1)
	Description    string `validate:"nonzero"`
	Metatitle      string
	Metakeywords   string
	Meta           string
}
type AuthResp struct {
	HasErrors bool
	Redirect  string
}

type Config struct {
	AdminUrl        string
	Email           string
	Password        string
	MaxItemsPerFile int
	Debug           bool
	SkipFirstRecord bool
}

type UploadResp struct {
	File struct {
		Name     string
		Type     string
		Tmp_name string
		Error    int
		Size     int
		Filename string
	}
}
