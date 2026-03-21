package httpapi

import (
	"net/http"
	"os"
	"strconv"

	"github.com/gonglijing/xunjiFsu/internal/service"
)

const errOnlyWasmFilesAllowedMessage = "Only .wasm files are allowed"

func (api *DriverAPI) UploadDriverFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		WriteBadRequest(w, "Failed to parse form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		WriteBadRequest(w, "Failed to get file: "+err.Error())
		return
	}
	defer file.Close()

	if !service.IsWasmFileName(header.Filename) {
		WriteBadRequest(w, errOnlyWasmFilesAllowedMessage)
		return
	}

	result, err := api.service.SaveUploadedDriver(header.Filename, header.Size, file)
	if err != nil {
		if os.IsPermission(err) {
			writeServerErrorWithLog(w, errSaveDriverFileFailed, err)
			return
		}
		if os.IsNotExist(err) {
			writeServerErrorWithLog(w, errCreateDriversDirFailed, err)
			return
		}
		writeServerErrorWithLog(w, errWriteDriverFileFailed, err)
		return
	}

	WriteSuccess(w, result)
}

func (api *DriverAPI) DownloadDriver(w http.ResponseWriter, r *http.Request) {
	driverModel, ok := api.loadDriverByRequest(w, r)
	if !ok {
		return
	}

	download, err := api.service.OpenDriverDownload(driverModel.ID)
	if err != nil {
		WriteNotFound(w, "File not found: "+err.Error())
		return
	}
	defer download.OpenFile.Close()

	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Disposition", "attachment; filename="+download.Name)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(download.Size, 10))

	http.ServeFile(w, r, download.Path)
}
