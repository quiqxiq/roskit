package services



func WrapPrintHTML(content string) string {
	return `<!DOCTYPE html>
<html>
<head>
<title>Voucher</title>
<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
<style>
body {
  color: #000000;
  background-color: #FFFFFF;
  font-size: 14px;
  font-family: 'Helvetica', arial, sans-serif;
  margin: 0px;
  -webkit-print-color-adjust: exact;
}
table.voucher {
  display: inline-block;
  border: 2px solid black;
  margin: 2px;
}
@page { size: auto; margin-left: 7mm; margin-right: 3mm; margin-top: 9mm; margin-bottom: 3mm; }
@media print {
  table { page-break-after:auto }
  tr    { page-break-inside:avoid; page-break-after:auto }
  td    { page-break-inside:avoid; page-break-after:auto }
  thead { display:table-header-group }
  tfoot { display:table-footer-group }
}
</style>
</head>
<body onload="window.print()">
` + content + `
</body>
</html>`
}
