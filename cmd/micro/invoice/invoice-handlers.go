package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/phpdave11/gofpdf"
	"github.com/phpdave11/gofpdf/contrib/gofpdi"
)

type Order struct {
	ID        int       `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"-"`
	Products  []Product `json:"products"`
}

type Product struct {
	Name     string
	Amount   int
	Quantity int
}

func (app *application) CreateAndSendInvoice(w http.ResponseWriter, r *http.Request) {
	// receive json
	var order Order

	err := app.readJSON(w, r, &order)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	// order.ID = 100
	// order.Email = "me@here.com"
	// order.FirstName = "John"
	// order.LastName = "Smith"
	// products := []Product{
	// 	{Name: "item", Amount: 1000, Quantity: 1},
	// 	{Name: "bottle", Amount: 1000, Quantity: 3},
	// 	{Name: "machine gun", Amount: 1000, Quantity: 1},
	// }

	// order.Products = products
	// order.CreatedAt = time.Now()

	// order.Quantity = 1
	// order.Amount = 1000
	// order.Product = "Item"

	// generate pdf invoice
	err = app.createInvoicePDF(order)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	// create mail

	attachments := []string{
		fmt.Sprintf("./invoices/%d.pdf", order.ID),
	}
	// send mail with attachment
	err = app.SendEmail("info@ecomm.com", order.Email, "Your invoice", "invoice", attachments, nil)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	// send response
	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	resp.Error = false
	resp.Message = fmt.Sprintf("Invoice %d.pdf created and sent to %s", order.ID, order.Email)
	app.writeJSON(w, http.StatusCreated, resp)
}

func (app *application) createInvoicePDF(order Order) error {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(10, 13, 10)
	pdf.SetAutoPageBreak(true, 0)

	importer := gofpdi.NewImporter()

	t := importer.ImportPage(pdf, "./pdf-templates/invoice.pdf", 1, "/MediaBox")

	pdf.AddPage()
	importer.UseImportedTemplate(pdf, t, 0, 0, 215.9, 0)

	// write info
	pdf.SetY(50)
	pdf.SetX(10)
	pdf.SetFont("Times", "", 11)
	pdf.CellFormat(97, 8, fmt.Sprintf("Attention: %s %s", order.FirstName, order.LastName), "", 0, "L", false, 0, "")

	pdf.Ln(5)
	pdf.CellFormat(97, 8, order.Email, "", 0, "L", false, 0, "")
	pdf.Ln(5)
	pdf.CellFormat(97, 8, order.CreatedAt.Format("01/02/2006"), "", 0, "L", false, 0, "")

	h := 0.0
	total := 0.0
	for _, product := range order.Products {

		pdf.SetX(58)
		pdf.SetY(93 + h)
		pdf.CellFormat(155, 8, product.Name, "", 0, "L", false, 0, "")
		pdf.SetX(166)
		pdf.CellFormat(20, 8, fmt.Sprintf("%d", product.Quantity), "", 0, "C", false, 0, "")

		pdf.SetX(185)
		pdf.CellFormat(20, 8, fmt.Sprintf("$%.2f", float32(product.Quantity*(product.Amount/100.00))), "", 0, "R", false, 0, "")
		// pdf.Ln(10)
		h += 5
		total += (float64(product.Quantity * (product.Amount / 100.00)))
	}
	pdf.SetY(240)
	pdf.SetX(185)
	pdf.CellFormat(20, 8, fmt.Sprintf("$%.2f", total), "", 0, "R", false, 0, "")

	invoicePath := fmt.Sprintf("./invoices/%d.pdf", order.ID)
	err := pdf.OutputFileAndClose(invoicePath)
	if err != nil {
		return err
	}
	return nil
}
