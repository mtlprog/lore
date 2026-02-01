package handler

import (
	"bytes"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/mtlprog/lore/internal/model"
	"github.com/mtlprog/lore/internal/service"
	"github.com/samber/lo"
)

// InitLanding handles GET /init - shows the landing page with account type selection.
func (h *Handler) InitLanding(w http.ResponseWriter, r *http.Request) {
	data := model.InitLandingData{
		Page: "landing",
	}

	var buf bytes.Buffer
	if err := h.tmpl.Render(&buf, "init.html", data); err != nil {
		slog.Error("failed to render init landing", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := buf.WriteTo(w); err != nil {
		slog.Debug("failed to write response", "error", err)
	}
}

// InitParticipant handles GET /init/participant - loads participant form.
func (h *Handler) InitParticipant(w http.ResponseWriter, r *http.Request) {
	accountID := r.URL.Query().Get("account_id")

	// If no account ID provided, show empty form
	if accountID == "" {
		h.renderParticipantForm(w, model.ParticipantFormData{}, "", "")
		return
	}

	// Validate account ID
	if err := service.ValidateAccountID(accountID); err != nil {
		h.renderParticipantForm(w, model.ParticipantFormData{}, "", "Invalid account ID format")
		return
	}

	// Fetch account from Horizon
	ctx := r.Context()
	acc, err := h.stellar.GetAccountDetail(ctx, accountID)
	if err != nil {
		if service.IsNotFound(err) {
			// Account doesn't exist - show empty form with just account ID
			form := model.ParticipantFormData{AccountID: accountID}
			original, _ := service.EncodeOriginalData(form)
			h.renderParticipantForm(w, form, original, "")
			return
		}
		slog.Error("failed to fetch account", "account_id", accountID, "error", err)
		h.renderParticipantForm(w, model.ParticipantFormData{}, "", "Failed to fetch account data")
		return
	}

	// Fetch raw account data for form parsing
	rawAcc, err := h.stellar.GetRawAccountData(ctx, accountID)
	if err != nil {
		slog.Error("failed to fetch raw account data", "account_id", accountID, "error", err)
		h.renderParticipantForm(w, model.ParticipantFormData{AccountID: accountID}, "", "")
		return
	}

	// Parse account data into form
	form := service.ParseAccountDataToParticipant(accountID, rawAcc)

	// Pre-fill with data from account detail if raw parsing missed something
	if form.Name == "" {
		form.Name = acc.Name
	}
	if form.About == "" {
		form.About = acc.About
	}
	if len(form.Tags) == 0 {
		// Filter to only available tags
		form.Tags = lo.Filter(acc.Tags, func(tag string, _ int) bool {
			return lo.Contains(model.AvailableTags, tag)
		})
	}

	original, _ := service.EncodeOriginalData(form)
	h.renderParticipantForm(w, form, original, "")
}

// InitParticipantSubmit handles POST /init/participant - form actions and preview.
func (h *Handler) InitParticipantSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	action := r.FormValue("action")
	current := h.parseParticipantForm(r)
	original := r.FormValue("original")

	switch action {
	case "add_partof":
		// Add empty PartOf field
		nextIndex := h.nextNumberedIndex(current.PartOf)
		current.PartOf = append(current.PartOf, model.NumberedField{
			Index: nextIndex,
			Value: "",
		})
		h.renderParticipantForm(w, current, original, "")

	case "remove_partof":
		// Remove PartOf field by index
		removeIdx, _ := strconv.Atoi(r.FormValue("remove_idx"))
		if removeIdx >= 0 && removeIdx < len(current.PartOf) {
			current.PartOf = append(current.PartOf[:removeIdx], current.PartOf[removeIdx+1:]...)
		}
		h.renderParticipantForm(w, current, original, "")

	case "preview":
		// Generate XDR preview
		h.previewParticipant(w, r, original, current)

	default:
		// Re-render form
		h.renderParticipantForm(w, current, original, "")
	}
}

// InitCorporate handles GET /init/corporate - loads corporate form.
func (h *Handler) InitCorporate(w http.ResponseWriter, r *http.Request) {
	accountID := r.URL.Query().Get("account_id")

	// If no account ID provided, show empty form
	if accountID == "" {
		h.renderCorporateForm(w, model.CorporateFormData{}, "", "")
		return
	}

	// Validate account ID
	if err := service.ValidateAccountID(accountID); err != nil {
		h.renderCorporateForm(w, model.CorporateFormData{}, "", "Invalid account ID format")
		return
	}

	// Fetch account from Horizon
	ctx := r.Context()
	acc, err := h.stellar.GetAccountDetail(ctx, accountID)
	if err != nil {
		if service.IsNotFound(err) {
			// Account doesn't exist - show empty form with just account ID
			form := model.CorporateFormData{AccountID: accountID}
			original, _ := service.EncodeOriginalData(form)
			h.renderCorporateForm(w, form, original, "")
			return
		}
		slog.Error("failed to fetch account", "account_id", accountID, "error", err)
		h.renderCorporateForm(w, model.CorporateFormData{}, "", "Failed to fetch account data")
		return
	}

	// Fetch raw account data for form parsing
	rawAcc, err := h.stellar.GetRawAccountData(ctx, accountID)
	if err != nil {
		slog.Error("failed to fetch raw account data", "account_id", accountID, "error", err)
		h.renderCorporateForm(w, model.CorporateFormData{AccountID: accountID}, "", "")
		return
	}

	// Parse account data into form
	form := service.ParseAccountDataToCorporate(accountID, rawAcc)

	// Pre-fill with data from account detail if raw parsing missed something
	if form.Name == "" {
		form.Name = acc.Name
	}
	if form.About == "" {
		form.About = acc.About
	}
	if len(form.Tags) == 0 {
		form.Tags = lo.Filter(acc.Tags, func(tag string, _ int) bool {
			return lo.Contains(model.AvailableTags, tag)
		})
	}

	original, _ := service.EncodeOriginalData(form)
	h.renderCorporateForm(w, form, original, "")
}

// InitCorporateSubmit handles POST /init/corporate - form actions and preview.
func (h *Handler) InitCorporateSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	action := r.FormValue("action")
	current := h.parseCorporateForm(r)
	original := r.FormValue("original")

	switch action {
	case "add_mypart":
		// Add empty MyPart field
		nextIndex := h.nextNumberedIndex(current.MyPart)
		current.MyPart = append(current.MyPart, model.NumberedField{
			Index: nextIndex,
			Value: "",
		})
		h.renderCorporateForm(w, current, original, "")

	case "remove_mypart":
		// Remove MyPart field by index
		removeIdx, _ := strconv.Atoi(r.FormValue("remove_idx"))
		if removeIdx >= 0 && removeIdx < len(current.MyPart) {
			current.MyPart = append(current.MyPart[:removeIdx], current.MyPart[removeIdx+1:]...)
		}
		h.renderCorporateForm(w, current, original, "")

	case "preview":
		// Generate XDR preview
		h.previewCorporate(w, r, original, current)

	default:
		// Re-render form
		h.renderCorporateForm(w, current, original, "")
	}
}

// renderParticipantForm renders the participant form template.
func (h *Handler) renderParticipantForm(w http.ResponseWriter, form model.ParticipantFormData, original, errorMsg string) {
	data := model.InitFormData{
		Page:          "participant",
		AccountID:     form.AccountID,
		FormData:      form,
		OriginalJSON:  original,
		AvailableTags: model.AvailableTags,
		Error:         errorMsg,
		FormAction:    "/init/participant",
		PreviewAction: "/init/participant",
	}

	var buf bytes.Buffer
	if err := h.tmpl.Render(&buf, "init.html", data); err != nil {
		slog.Error("failed to render participant form", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := buf.WriteTo(w); err != nil {
		slog.Debug("failed to write response", "error", err)
	}
}

// renderCorporateForm renders the corporate form template.
func (h *Handler) renderCorporateForm(w http.ResponseWriter, form model.CorporateFormData, original, errorMsg string) {
	data := model.InitFormData{
		Page:          "corporate",
		AccountID:     form.AccountID,
		FormData:      form,
		OriginalJSON:  original,
		AvailableTags: model.AvailableTags,
		Error:         errorMsg,
		FormAction:    "/init/corporate",
		PreviewAction: "/init/corporate",
	}

	var buf bytes.Buffer
	if err := h.tmpl.Render(&buf, "init.html", data); err != nil {
		slog.Error("failed to render corporate form", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := buf.WriteTo(w); err != nil {
		slog.Debug("failed to write response", "error", err)
	}
}

// previewParticipant generates and displays XDR preview for participant form.
func (h *Handler) previewParticipant(w http.ResponseWriter, r *http.Request, originalEncoded string, current model.ParticipantFormData) {
	original, err := service.DecodeOriginalParticipant(originalEncoded)
	if err != nil {
		h.renderParticipantForm(w, current, originalEncoded, "Failed to decode form state")
		return
	}

	// Fetch current sequence number from network (transaction needs current + 1)
	ctx := r.Context()
	seqNum, err := h.stellar.GetAccountSequence(ctx, current.AccountID)
	if err != nil {
		h.renderParticipantForm(w, current, originalEncoded, "Failed to fetch account sequence number")
		return
	}

	builder := service.NewInitXDRBuilder()
	xdr, ops, err := builder.GenerateParticipantXDR(original, current, seqNum+1)
	if err != nil {
		h.renderParticipantForm(w, current, originalEncoded, err.Error())
		return
	}

	data := model.InitPreviewData{
		Page:       "preview",
		AccountID:  current.AccountID,
		XDR:        xdr,
		Operations: ops,
		LabLink:    service.BuildLabLink(xdr),
	}

	var buf bytes.Buffer
	if err := h.tmpl.Render(&buf, "init.html", data); err != nil {
		slog.Error("failed to render preview", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := buf.WriteTo(w); err != nil {
		slog.Debug("failed to write response", "error", err)
	}
}

// previewCorporate generates and displays XDR preview for corporate form.
func (h *Handler) previewCorporate(w http.ResponseWriter, r *http.Request, originalEncoded string, current model.CorporateFormData) {
	original, err := service.DecodeOriginalCorporate(originalEncoded)
	if err != nil {
		h.renderCorporateForm(w, current, originalEncoded, "Failed to decode form state")
		return
	}

	// Fetch current sequence number from network (transaction needs current + 1)
	ctx := r.Context()
	seqNum, err := h.stellar.GetAccountSequence(ctx, current.AccountID)
	if err != nil {
		h.renderCorporateForm(w, current, originalEncoded, "Failed to fetch account sequence number")
		return
	}

	builder := service.NewInitXDRBuilder()
	xdr, ops, err := builder.GenerateCorporateXDR(original, current, seqNum+1)
	if err != nil {
		h.renderCorporateForm(w, current, originalEncoded, err.Error())
		return
	}

	data := model.InitPreviewData{
		Page:       "preview",
		AccountID:  current.AccountID,
		XDR:        xdr,
		Operations: ops,
		LabLink:    service.BuildLabLink(xdr),
	}

	var buf bytes.Buffer
	if err := h.tmpl.Render(&buf, "init.html", data); err != nil {
		slog.Error("failed to render preview", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := buf.WriteTo(w); err != nil {
		slog.Debug("failed to write response", "error", err)
	}
}

// maxNumberedFields is the maximum number of numbered fields allowed per form.
const maxNumberedFields = 100

// parseParticipantForm extracts ParticipantFormData from HTTP form values.
func (h *Handler) parseParticipantForm(r *http.Request) model.ParticipantFormData {
	form := model.ParticipantFormData{
		AccountID: r.FormValue("account_id"),
		Name:      r.FormValue("name"),
		About:     r.FormValue("about"),
		Website:   r.FormValue("website"),
		Tags:      r.Form["tags"],
	}

	// Parse PartOf fields with limit to prevent DoS
	for i := 0; i < maxNumberedFields; i++ {
		indexKey := "partof_index_" + strconv.Itoa(i)
		valueKey := "partof_value_" + strconv.Itoa(i)

		index := r.FormValue(indexKey)
		value := r.FormValue(valueKey)

		if index == "" && value == "" {
			break
		}

		form.PartOf = append(form.PartOf, model.NumberedField{
			Index: index,
			Value: value,
		})
	}

	return form
}

// parseCorporateForm extracts CorporateFormData from HTTP form values.
func (h *Handler) parseCorporateForm(r *http.Request) model.CorporateFormData {
	form := model.CorporateFormData{
		AccountID: r.FormValue("account_id"),
		Name:      r.FormValue("name"),
		About:     r.FormValue("about"),
		Website:   r.FormValue("website"),
		Tags:      r.Form["tags"],
	}

	// Parse MyPart fields with limit to prevent DoS
	for i := 0; i < maxNumberedFields; i++ {
		indexKey := "mypart_index_" + strconv.Itoa(i)
		valueKey := "mypart_value_" + strconv.Itoa(i)

		index := r.FormValue(indexKey)
		value := r.FormValue(valueKey)

		if index == "" && value == "" {
			break
		}

		form.MyPart = append(form.MyPart, model.NumberedField{
			Index: index,
			Value: value,
		})
	}

	return form
}

// nextNumberedIndex returns the next available index for numbered fields.
func (h *Handler) nextNumberedIndex(fields []model.NumberedField) string {
	maxNum := 0
	for _, f := range fields {
		if num, err := strconv.Atoi(f.Index); err == nil && num >= maxNum {
			maxNum = num + 1
		}
	}
	return strconv.Itoa(maxNum)
}
