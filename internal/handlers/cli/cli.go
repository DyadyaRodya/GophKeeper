package cli

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
	"github.com/DyadyaRodya/GophKeeper/internal/usecases/dto"
	"github.com/DyadyaRodya/GophKeeper/pkg/files"
)

type HandlerCode int

const (
	HandlersProcessError HandlerCode = iota - 1
	HandlersExiting
	HandlersOk
	HandlersNeedLogin

	skipSyncCount = 10
)

type (
	ListDataUsecase interface {
		Handle(ctx context.Context, userUUID string) ([]*domainmodels.DataInfo, error)
	}
	LoginUsecase interface {
		HandleOnline(
			ctx context.Context,
			username string,
			password string,
		) (*dto.ClientLoginResult, error)
		HandleOffline(ctx context.Context, password string) (*dto.ClientLoginResult, error)
	}
	ReadDataUsecase interface {
		Handle(
			ctx context.Context,
			dek []byte,
			userUUID string,
			dataUUID string,
		) (*domainmodels.DataInfo, *domainmodels.RawData, error)
	}
	RecoverUsecase interface {
		Handle(
			ctx context.Context,
			username string,
			recoveryKey []byte,
			newPassword string,
		) (*dto.ClientRecoverResult, error)
	}
	RegisterUsecase interface {
		Handle(
			ctx context.Context,
			username string,
			password string,
		) (*dto.ClientRegisterResult, error)
	}
	SaveDataUsecase interface {
		HandleAdd(
			ctx context.Context,
			dek []byte,
			userUUID string,
			rawData *domainmodels.RawData,
		) (*domainmodels.DataInfo, error)
		HandleUpdate(
			ctx context.Context,
			dek []byte,
			userUUID string,
			dataUUID string,
			rawData *domainmodels.RawData,
		) (*domainmodels.DataInfo, error)
	}
	DeleteDataUsecase interface {
		Handle(
			ctx context.Context,
			userUUID string,
			dataUUID string,
		) error
	}
	UpdatePasswordUsecase interface {
		Handle(
			ctx context.Context,
			oldPassword,
			newPassword string,
		) (*dto.ClientRecoverResult, error)
	}
	SyncDataUsecase interface {
		Handle(ctx context.Context, userUUID string) error
	}
	// Handlers handles user interaction on client side
	Handlers struct {
		ListDataUsecase       ListDataUsecase
		LoginUsecase          LoginUsecase
		ReadDataUsecase       ReadDataUsecase
		RecoverUsecase        RecoverUsecase
		RegisterUsecase       RegisterUsecase
		SaveDataUsecase       SaveDataUsecase
		DeleteDataUsecase     DeleteDataUsecase
		UpdatePasswordUsecase UpdatePasswordUsecase
		SyncDataUsecase       SyncDataUsecase
		appLogger             *zap.Logger
		dek                   []byte
		userUUID              string
		offlineLogin          bool
		debug                 bool
		synced                bool
		syncSkippedCounter    int
	}
)

// NewHandlers constructor for Handlers
func NewHandlers(
	listDataUsecase ListDataUsecase,
	loginUsecase LoginUsecase,
	readDataUsecase ReadDataUsecase,
	recoverUsecase RecoverUsecase,
	registerUsecase RegisterUsecase,
	saveDataUsecase SaveDataUsecase,
	deleteDataUsecase DeleteDataUsecase,
	updatePasswordUsecase UpdatePasswordUsecase,
	syncDataUsecase SyncDataUsecase,
	appLogger *zap.Logger,
	debug bool,
) *Handlers {
	return &Handlers{
		ListDataUsecase:       listDataUsecase,
		LoginUsecase:          loginUsecase,
		ReadDataUsecase:       readDataUsecase,
		RecoverUsecase:        recoverUsecase,
		RegisterUsecase:       registerUsecase,
		SaveDataUsecase:       saveDataUsecase,
		UpdatePasswordUsecase: updatePasswordUsecase,
		DeleteDataUsecase:     deleteDataUsecase,
		SyncDataUsecase:       syncDataUsecase,
		appLogger:             appLogger,
		debug:                 debug,
		// fill after Login|Register|Recovery
		dek:                nil,
		userUUID:           "",
		offlineLogin:       false,
		synced:             false,
		syncSkippedCounter: 0,
	}
}

func (h *Handlers) EnterMenu(ctx context.Context) (HandlerCode, error) {
	for {
		select {
		case <-ctx.Done():
			// Context was canceled, return the context error
			return HandlersExiting, ctx.Err()
		default:

		}
		h.synced = false
		fmt.Println("\nThis is entering menu. Choose an option:")
		fmt.Println("1. Login")
		fmt.Println("2. Register")
		fmt.Println("3. Recover Password")
		fmt.Println("0. Exit")
		fmt.Print("Enter your choice: ")

		choice := h.getUserInput()

		switch choice {
		case "1":
			username, password := h.getCredentials()
			h.offlineLogin = false
			res, err := h.LoginUsecase.HandleOnline(ctx, username, password)
			if err != nil && errors.Is(err, domainmodels.ErrOffline) {
				fmt.Println("Server unavailable. " +
					"Tying offline work. If connection restores, you'll be asked to enter again.")
				h.offlineLogin = true
				res, err = h.LoginUsecase.HandleOffline(ctx, password)
			}
			if err != nil {
				h.printLoginError(err)
				continue
			}
			h.dek = res.DEK
			h.userUUID = res.UserUUID
			fmt.Println("Success.")
		case "2":
			username, password := h.getCredentials()
			res, err := h.RegisterUsecase.Handle(ctx, username, password)
			if err != nil {
				h.printRegisterError(err)
				continue
			}
			h.dek = res.DEK
			h.userUUID = res.UserUUID
			h.offlineLogin = false
			fmt.Println("Success. Save your recovery key (you will not be able to see it again):")
			fmt.Println(hex.EncodeToString(res.RecoveryKey))
		case "3":
			username, code, newPassword := h.getRecoveryDetails()
			recoveryKey, err := hex.DecodeString(code)
			if err != nil {
				fmt.Println("Error decoding recovery key. Make sure you entered correct code.")
			}
			res, err := h.RecoverUsecase.Handle(ctx, username, recoveryKey, newPassword)
			if err != nil {
				h.printRecoverError(err)
				continue
			}
			h.dek = res.DEK
			h.userUUID = res.UserUUID
			h.offlineLogin = false
			fmt.Println("Success. Save your new recovery key (you will not be able to see it again):")
			fmt.Println(hex.EncodeToString(res.RecoveryKey))
		case "0":
			fmt.Println("Exiting...")
			return HandlersExiting, nil
		default:
			fmt.Println("Invalid choice, please try again.")
			continue
		}

		code, err := h.postenterMenu(ctx)
		if code == HandlersExiting {
			return HandlersExiting, err
		}
		if err != nil || code == HandlersProcessError {
			fmt.Println("Error occurred. Use debug to see details.")
			h.printDebug(err)
			return HandlersProcessError, err
		}
		if code == HandlersNeedLogin {
			fmt.Println("Need to enter again.")
		}
	}
}

func (h *Handlers) postenterMenu(ctx context.Context) (HandlerCode, error) {
	for {

		select {
		case <-ctx.Done():
			// Context was canceled, return the context error
			return HandlersExiting, ctx.Err()
		default:

		}
		fmt.Println("\nMenu:")
		fmt.Println("1. Update Password")
		fmt.Println("2. Work with Data")
		fmt.Println("9. Back to entering menu")
		fmt.Println("0. Exit")
		fmt.Print("Enter your choice: ")

		choice := h.getUserInput()

		switch choice {
		case "1":
			if h.offlineLogin {
				fmt.Println("You was offline. Need to login first.")
				fmt.Println("Go to previous menu to login or recover password.")
				fmt.Println("Or you can continue working offline until you asked to enter.")
				continue
			}
			h.updatePassword(ctx)
		case "2":
			code, err := h.workWithData(ctx)
			if code == HandlersExiting {
				return HandlersExiting, err
			}
			if err != nil || code == HandlersProcessError {
				fmt.Println("Error occurred. Use debug to see details.")
				h.printDebug(err)
				return HandlersProcessError, err
			}
			if code == HandlersNeedLogin {
				return HandlersNeedLogin, nil
			}
		case "9":
			fmt.Println("Returning to previous menu...")
			return HandlersOk, nil
		case "0":
			fmt.Println("Exiting...")
			return HandlersExiting, nil
		default:
			fmt.Println("Invalid choice, please try again.")
			continue
		}
	}
}

func (h *Handlers) updatePassword(ctx context.Context) {
	fmt.Print("Enter current password: ")
	currentPassword := h.getUserInput()
	fmt.Print("Enter new password: ")
	newPassword := h.getUserInput()
	res, err := h.UpdatePasswordUsecase.Handle(ctx, currentPassword, newPassword)
	if err != nil {
		switch {
		case errors.Is(err, domainmodels.ErrWrongCredentials):
			fmt.Println("Make sure you entered correct password and try again. Or recover password.")
		case errors.Is(err, domainmodels.ErrPasswordComplexity):
			fmt.Printf("Validation password: %s\n", err.Error())
		case errors.Is(err, domainmodels.ErrGateway):
			fmt.Println("Server unavailable or server error. Try again later.")
		default:
			fmt.Println("Error occurred. Try again later or use debug to see more details.")
		}
		h.printDebug(err)
	}
	h.dek = res.DEK
	h.userUUID = res.UserUUID
	fmt.Println("Success. Save your new recovery key (you will not be able to see it again):")
	fmt.Println(hex.EncodeToString(res.RecoveryKey))
}

func (h *Handlers) workWithData(ctx context.Context) (HandlerCode, error) {
	for {
		select {
		case <-ctx.Done():
			// Context was canceled, return the context error
			return HandlersExiting, ctx.Err()
		default:

		}
		if h.syncData(ctx) == HandlersNeedLogin {
			return HandlersNeedLogin, nil
		}

		fmt.Println("\nData Management:")
		fmt.Println("1. List Stored Data")
		fmt.Println("2. Read Some Data")
		fmt.Println("3. Save New Data")
		fmt.Println("4. Update Existing Data")
		fmt.Println("5. Delete Some Data")
		fmt.Println("9. Go Back to Previous Menu")
		fmt.Println("0. Exit")
		fmt.Print("Enter your choice: ")

		choice := h.getUserInput()
		var code HandlerCode
		switch choice {
		case "1":
			code = h.listStoredData(ctx)
		case "2":
			code = h.readSomeData(ctx)
		case "3":
			code = h.saveNewData(ctx)

		case "4":
			code = h.updateExistingData(ctx)
		case "5":
			code = h.deleteSomeData(ctx)
		case "9":
			fmt.Println("Returning to previous menu...")
			return HandlersOk, nil
		case "0":
			fmt.Println("Exiting...")
			return HandlersExiting, nil
		default:
			fmt.Println("Invalid choice, please try again.")
			continue
		}

		if code == HandlersProcessError {
			fmt.Println("Error occurred. Try again later or use debug to see more details.")
			continue
		}
		if code == HandlersExiting {
			return HandlersExiting, ctx.Err()
		}
		if code != HandlersOk {
			return code, nil
		}
	}
}

func (h *Handlers) syncData(ctx context.Context) HandlerCode {
	if h.offlineLogin || h.synced && h.syncSkippedCounter < skipSyncCount {
		h.syncSkippedCounter++
		return HandlersOk
	}
	h.syncSkippedCounter = 0
	fmt.Println("Going to sync data...")
	err := h.SyncDataUsecase.Handle(ctx, h.userUUID)
	if err != nil {
		h.printDebug(err)
		switch {
		case errors.Is(err, domainmodels.ErrWrongCredentials):
			return HandlersNeedLogin
		case errors.Is(err, domainmodels.ErrOffline):
			return HandlersOk // try again later
		default:
			fmt.Println("Error occurred while sync data. Use debug to see details.")
		}
	} else {
		h.synced = true
	}
	return HandlersOk
}

func (h *Handlers) listStoredData(ctx context.Context) HandlerCode {
	res, err := h.ListDataUsecase.Handle(ctx, h.userUUID)
	if err != nil {
		switch {
		case errors.Is(err, domainmodels.ErrWrongCredentials):
			return HandlersNeedLogin
		case errors.Is(err, domainmodels.ErrGateway):
			h.synced = false
			fmt.Println("Server server error. Try again later.")
			return HandlersOk
		default:
			fmt.Println("Error occurred. Try again later or use debug to see more details.")
			h.printDebug(err)
			return HandlersOk
		}
	}
	code := HandlersOk
	for _, data := range res {
		fmt.Printf("UUID: %s, Last updated: %s, was deleted: %v\n",
			data.UUID, data.LastUpdated.Local().String(), data.IsDeleted)
		if data.IsDeleted {
			if h.debug {
				fmt.Println("Data was deleted. Trying to sync data with server...")
			}
			newCode := h.handleDeleted(ctx, data.UUID)
			if newCode == HandlersProcessError {
				code = HandlersProcessError
			} else if newCode == HandlersExiting && code != HandlersProcessError {
				code = HandlersExiting
			} else if newCode == HandlersNeedLogin && code == HandlersOk {
				code = HandlersNeedLogin
			}
		}
	}
	return code
}

func (h *Handlers) readSomeData(ctx context.Context) HandlerCode {
	fmt.Print("Enter UUID of the data to read: ")
	uuid := h.getUserInput()
	meta, data, err := h.ReadDataUsecase.Handle(ctx, h.dek, h.userUUID, uuid)
	if err != nil {
		switch {
		case errors.Is(err, domainmodels.ErrDataInfoNotFound):
			fmt.Println("Data not found. Check uuid and try again.")
			return HandlersOk
		case errors.Is(err, domainmodels.ErrDataDeleted):
			fmt.Println("Data was deleted. Trying to sync some data with server...")
			return h.handleDeleted(ctx, meta.UUID) // not full sync, deleted only
		case errors.Is(err, domainmodels.ErrWrongCredentials):
			return HandlersNeedLogin
		case errors.Is(err, domainmodels.ErrGateway):
			h.synced = false
			fmt.Println("Server server error. Try again later.")
			return HandlersOk
		default:
			fmt.Println("Error occurred. Try again later or use debug to see more details.")
			h.printDebug(err)
			return HandlersOk
		}
	}
	fmt.Printf("UUID: %s, Last updated: %s, was deleted: %v\n",
		meta.UUID, meta.LastUpdated.Local().String(), meta.IsDeleted)

	return h.processData(ctx, data)
}

func (h *Handlers) handleDeleted(ctx context.Context, uuid string) HandlerCode {
	err := h.DeleteDataUsecase.Handle(ctx, h.userUUID, uuid)
	if err != nil && !errors.Is(err, domainmodels.ErrWrongCredentials) {
		fmt.Println("Error occurred. Try again later or use debug to see more details.")
		h.printDebug(err)
	} else if errors.Is(err, domainmodels.ErrWrongCredentials) {
		return HandlersNeedLogin
	}
	return HandlersOk
}

func (h *Handlers) saveNewData(ctx context.Context) HandlerCode {
	rawData, code := h.getRawData(ctx)
	if code != HandlersOk {
		return code
	}
	meta, err := h.SaveDataUsecase.HandleAdd(ctx, h.dek, h.userUUID, rawData)
	if err != nil {
		return h.printSaveDataError(err)
	}
	fmt.Printf("Data saved successfully. UUID:%s\n", meta.UUID)
	return HandlersOk
}

func (h *Handlers) printSaveDataError(err error) HandlerCode {
	h.printDebug(err)
	switch {
	case errors.Is(err, domainmodels.ErrOffline):
		fmt.Println("Saving locally.")
		h.synced = false
	case errors.Is(err, domainmodels.ErrWrongCredentials):
		return HandlersNeedLogin
	case errors.Is(err, domainmodels.ErrGateway):
		fmt.Println("Server error. Try again later.")
		h.synced = false
	default:
		fmt.Println("Error occurred. Try again later or use debug to see more details.")
	}
	return HandlersOk
}

func (h *Handlers) updateExistingData(ctx context.Context) HandlerCode {
	fmt.Print("Enter UUID of the data to update: ")
	uuid := h.getUserInput()
	rawData, code := h.getRawData(ctx)
	if code != HandlersOk {
		return code
	}
	meta, err := h.SaveDataUsecase.HandleUpdate(ctx, h.dek, h.userUUID, uuid, rawData)
	if err != nil {
		return h.printSaveDataError(err)
	}
	fmt.Printf("Data updated successfully. UUID:%s\n", meta.UUID)
	return HandlersOk
}

func (h *Handlers) getRawData(ctx context.Context) (*domainmodels.RawData, HandlerCode) {
	rawdata := &domainmodels.RawData{}
	for {
		select {
		case <-ctx.Done():
			// Context was canceled
			return nil, HandlersExiting
		default:

		}
		fmt.Printf("Enter data type (%s/%s/%s/%s) or command (9. Go Back, 0. Exit): ",
			string(domainmodels.CredsDataType),
			string(domainmodels.CardDataType),
			string(domainmodels.TextDataType),
			string(domainmodels.BinaryDataType))
		choice := h.getUserInput()
		switch choice {
		case "9":
			fmt.Println("Returning to previous menu...")
			return nil, HandlersOk
		case "0":
			fmt.Println("Exiting...")
			return nil, HandlersExiting
		default:

		}
		dataType := domainmodels.DataType(choice)
		switch dataType {
		case domainmodels.CredsDataType:
			username, password := h.getCredentials()
			rawdata.Creds = &domainmodels.CredsData{
				Login: username,
				Pass:  password,
			}
		case domainmodels.CardDataType:
			fmt.Print("Enter Number: ")
			number := h.getUserInput()
			fmt.Print("Enter Date: ")
			data := h.getUserInput()
			fmt.Print("Enter Code: ")
			cardCode := h.getUserInput()
			fmt.Print("Enter Holder: ")
			holder := h.getUserInput()
			rawdata.Card = &domainmodels.CardData{
				Number: number,
				Date:   data,
				Code:   cardCode,
				Holder: holder,
			}
		case domainmodels.TextDataType:
			fmt.Print("Enter file path: ")
			filePath := h.getUserInput()
			text, code := h.readTextFile(ctx, filePath)
			if code != HandlersOk {
				return nil, code
			}
			rawdata.Text = &domainmodels.TextData{
				Data: text,
			}
		case domainmodels.BinaryDataType:
			fmt.Print("Enter file path: ")
			filePath := h.getUserInput()
			data, code := h.readBinaryFile(ctx, filePath)
			if code != HandlersOk {
				return nil, code
			}
			rawdata.Binary = &domainmodels.BinaryData{
				Data: data,
			}
		default:
			fmt.Println("Invalid choice, please try again.")
			continue
		}
		rawdata.DataType = dataType
		break
	}
	fmt.Println("Enter additional data (enter '\\q' on a new line to finish):")
	rawdata.MetaInfo = h.readMultiLineInput()
	return rawdata, HandlersOk
}

func (h *Handlers) deleteSomeData(ctx context.Context) HandlerCode {
	fmt.Print("Enter UUID of the data to delete: ")
	uuid := h.getUserInput()
	code := h.handleDeleted(ctx, uuid)
	if code == HandlersOk {
		fmt.Println("Data deleted successfully.")
	}
	return code
}

func (h *Handlers) processData(ctx context.Context, data *domainmodels.RawData) HandlerCode {
	fmt.Printf("Type: %s, Meta: %s\n", data.DataType, data.MetaInfo)
	if data.DataType == domainmodels.TextDataType || data.DataType == domainmodels.BinaryDataType {
		fmt.Printf("Enter file path to save %s data: ", string(data.DataType))
		filePath := h.getUserInput()
		return h.processFileData(ctx, filePath, data)
	}

	if data.DataType == domainmodels.CredsDataType {
		fmt.Printf("Login: %s, Password: %s\n", data.Creds.Login, data.Creds.Pass)
	} else {
		fmt.Printf("Number: %s, Date: %s, Code: %s, Holder: %s\n",
			data.Card.Number, data.Card.Date, data.Card.Code, data.Card.Holder)
	}
	return HandlersOk
}

func (h *Handlers) processFileData(ctx context.Context, filePath string, data *domainmodels.RawData) HandlerCode {
	writeError := false
	for {
		select {
		case <-ctx.Done():
			// Context was canceled, return the context error
			return HandlersExiting
		default:

		}
		if writeError {
			fmt.Println("Choose option")
			fmt.Println("1. Enter different file path")
			fmt.Println("9. Go back")
			fmt.Println("0. Exit")
			fmt.Print("Enter your choice: ")

			choice := h.getUserInput()
			switch choice {
			case "1":
				fmt.Printf("Enter file path to save %s data: ", string(data.DataType))
				filePath = h.getUserInput()
				writeError = false
			case "9":
				fmt.Println("Returning to previous menu...")
				return HandlersOk
			case "0":
				fmt.Println("Exiting...")
				return HandlersExiting
			default:
				fmt.Println("Invalid choice, please try again.")
				continue
			}
		}

		if files.FileExists(filePath) {
			fmt.Printf("File %s exists. Rewrite?\n", filePath)
			fmt.Println("1. Rewrite")
			fmt.Println("2. Enter different file path")
			fmt.Println("9. Go back")
			fmt.Println("0. Exit")
			fmt.Print("Enter your choice: ")

			choice := h.getUserInput()
			switch choice {
			case "1":
				// process outside if statement
			case "2":
				fmt.Printf("Enter file path to save %s data: ", string(data.DataType))
				filePath = h.getUserInput()
			case "9":
				fmt.Println("Returning to previous menu...")
				return HandlersOk
			case "0":
				fmt.Println("Exiting...")
				return HandlersExiting
			default:
				fmt.Println("Invalid choice, please try again.")
				continue
			}
		}
		code := h.writeFile(ctx, filePath, data)
		if code != HandlersProcessError {
			return code
		}
		writeError = true
	}
}

func (h *Handlers) writeFile(ctx context.Context, filePath string, data *domainmodels.RawData) HandlerCode {
	dataFile, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Error opening file: %s", err.Error())
		return HandlersProcessError
	}
	defer dataFile.Close()

	done := make(chan error, 1)
	go func() {
		fmt.Println("Writing")
		if data.DataType == domainmodels.BinaryDataType {
			_, err = dataFile.Write(data.Binary.Data)
			done <- err
		} else {
			_, err = dataFile.WriteString(data.Text.Data)
			done <- err
		}
	}()
	// Wait for either write to complete or the context to be canceled
	select {
	case <-ctx.Done():
		// Context was canceled, return the context error
		return HandlersExiting
	case err = <-done:
		// Write operation completed, return its result
	}
	if err != nil {
		fmt.Printf("Error writing to file: %s", err.Error())
		return HandlersProcessError
	}
	fmt.Printf("Data saved to %s\n", filePath)
	return HandlersOk
}

func (h *Handlers) readTextFile(ctx context.Context, filePath string) (string, HandlerCode) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening file: %s", err.Error())
		return "", HandlersProcessError
	}
	defer file.Close()

	var content strings.Builder
	scanner := bufio.NewScanner(file)

	done := make(chan error, 1)
	go func() {
		for scanner.Scan() {
			content.WriteString(scanner.Text() + "\n") // Append each line to the string
		}
		done <- nil
	}()

	// Wait for either write to complete or the context to be canceled
	select {
	case <-ctx.Done():
		// Context was canceled, return the context error
		return "", HandlersExiting
	case <-done:
		// Write operation completed, return its result
	}
	if err = scanner.Err(); err != nil {
		fmt.Printf("Error scanning file: %s", err.Error())
		return "", HandlersProcessError
	}

	return content.String(), HandlersOk
}

func (h *Handlers) readBinaryFile(ctx context.Context, filePath string) ([]byte, HandlerCode) {
	var data []byte
	done := make(chan error, 1)
	go func() {
		var err error
		data, err = os.ReadFile(filePath)
		done <- err
	}()

	// Wait for either write to complete or the context to be canceled
	select {
	case <-ctx.Done():
		// Context was canceled, return the context error
		return nil, HandlersExiting
	case err := <-done:
		// Write operation completed, return its result
		if err != nil {
			fmt.Printf("Error scanning file: %s", err.Error())
			return nil, HandlersProcessError
		}
	}

	return data, HandlersOk
}

func (h *Handlers) printDebug(err error) {
	if h.debug && err != nil {
		h.appLogger.Debug("Error occurred", zap.Error(err))
	} else if h.debug {
		h.appLogger.Debug("No error info here")
	}
}

func (h *Handlers) printLoginError(err error) {
	switch {
	case errors.Is(err, domainmodels.ErrUserKeysNotFound):
		fmt.Println("Local session keys not found. Check connection and try again.")
	case errors.Is(err, domainmodels.ErrWrongCredentials):
		fmt.Println("Wrong credentials. Please try again or choose another option.")
	case errors.Is(err, domainmodels.ErrGateway):
		fmt.Println("Server error. Try again later.")
	default:
		fmt.Println("Cannot decrypt keys. " +
			"Probably password wrong or local session belongs to other user. " +
			"Check connection and try login again with other credentials.")
	}
	h.printDebug(err)
}

func (h *Handlers) printRegisterError(err error) {
	switch {
	case errors.Is(err, domainmodels.ErrLoginTaken):
		fmt.Println("Username taken. Try other username, enter or recover password.")
	case errors.Is(err, domainmodels.ErrLoginValidation) || errors.Is(err, domainmodels.ErrPasswordComplexity):
		fmt.Printf("Validation credentials: %s\n", err.Error())
	case errors.Is(err, domainmodels.ErrGateway):
		fmt.Println("Server unavailable or server error. Try again later.")
	default:
		fmt.Println("Error occurred. Try again later or use debug to see more details.")
	}
	h.printDebug(err)
}

func (h *Handlers) printRecoverError(err error) {
	switch {
	case errors.Is(err, domainmodels.ErrWrongCredentials):
		fmt.Println("Make sure you entered correct username and code and try again.")
	case errors.Is(err, domainmodels.ErrPasswordComplexity):
		fmt.Printf("Validation password: %s\n", err.Error())
	case errors.Is(err, domainmodels.ErrGateway):
		fmt.Println("Server unavailable or server error. Try again later.")
	default:
		fmt.Println("Error occurred. Try again later or use debug to see more details.")
	}
	h.printDebug(err)
}

func (h *Handlers) getOldNewPassword() (string, string) {
	fmt.Print("Enter old password: ")
	oldPassword := h.getUserInput()
	fmt.Print("Enter new password: ")
	newPassword := h.getUserInput()
	return oldPassword, newPassword
}

func (h *Handlers) getCredentials() (string, string) {
	fmt.Print("Enter username: ")
	username := h.getUserInput()
	fmt.Print("Enter password: ")
	password := h.getUserInput()
	return username, password
}

func (h *Handlers) getRecoveryDetails() (string, string, string) {
	fmt.Print("Enter username: ")
	username := h.getUserInput()
	fmt.Print("Enter recovery code: ")
	code := h.getUserInput()
	fmt.Print("Enter new password: ")
	newPassword := h.getUserInput()
	return username, code, newPassword
}

func (h *Handlers) getUserInput() string {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func (h *Handlers) readMultiLineInput() string {
	var lines []string
	scanner := bufio.NewScanner(os.Stdin)
	for {
		scanner.Scan()
		line := scanner.Text()
		if line == "\\q" {
			break
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
