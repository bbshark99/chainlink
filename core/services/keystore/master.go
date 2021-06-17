package keystore

// func newV1(db *gorm.DB, config orm.ConfigReader) *master {
// 	scryptParams := utils.GetScryptParams(config)
// 	masterV2 := newV2(db, config)
// 	return &master{
// 		eth: newEthKeyStore(db, scryptParams),
// 		csa: masterV2,
// 		ocr: newOCRKeyStore(db, scryptParams),
// 		vrf: newVRFKeyStore(db, scryptParams),
// 	}
// }

// type master struct {
// 	eth Eth
// 	csa CSA
// 	ocr OCR
// 	vrf VRF
// }

// func (m *master) Eth() Eth {
// 	return m.eth
// }

// func (m *master) CSA() CSA {
// 	return m.csa
// }

// func (m *master) OCR() OCR {
// 	return m.ocr
// }

// func (m *master) VRF() VRF {
// 	return m.vrf
// }

// func (m *master) Unlock(string) error {
// 	return nil
// }
