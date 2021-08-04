// Package gethwrappers keeps track of the golang wrappers of the solidity contracts
package gethwrappers

//go:generate ./compile.sh 15000 ../../../../libocr-internal/contract2/OCRTitleRequest.sol

//go:generate ./compile.sh 15000 ../../../../libocr-internal/contract2/AccessControlledOffchainAggregator.sol
//go:generate ./compile.sh 15000 ../../../../libocr-internal/contract2/OffchainAggregator.sol
//go:generate ./compile.sh 15000 ../../../../libocr-internal/contract2/ExposedOffchainAggregator.sol

//go:generate ./compile.sh 1000 ../../../../libocr-internal/contract2/TestOffchainAggregator.sol
//go:generate ./compile.sh 1000 ../../../../libocr-internal/contract2/TestValidator.sol
//go:generate ./compile.sh 1000 ../../../../libocr-internal/contract2/AccessControlTestHelper.sol
