package service

import (
	"fmt"
	"net/http"
)

func DeleteConvene(apiUrl, token, gameUid string) error {
	url := fmt.Sprintf("%s/convene/delete/%s", apiUrl, gameUid)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete convene failed, status: %d", resp.StatusCode)
	}
	return nil
}
