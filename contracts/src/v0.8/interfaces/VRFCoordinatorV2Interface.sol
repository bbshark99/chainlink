// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface VRFCoordinatorV2Interface {

  function requestRandomWords(
    bytes32 keyHash,  // Corresponds to a particular offchain job which uses that key for the proofs
    uint64  subId,   // A data structure for billing
    uint16  minimumRequestConfirmations,
    uint32  callbackGasLimit,
    uint32  numWords  // Desired number of random words
  )
    external
    returns (
      uint256 requestId
    );

  function createSubscription(
    address[] memory consumers // permitted consumers of the subscription
  )
    external
    returns (
      uint64 subId
    );

  function getSubscription(
    uint64 subId
  )
    external
    view
    returns (
      uint96 balance,
      address owner,
      address[] memory consumers
    );

  function requestSubscriptionOwnerTransfer(
    uint64 subId,
    address newOwner
  )
    external;

  function acceptSubscriptionOwnerTransfer(
    uint64 subId
  )
    external;

  function addConsumer(
    uint64 subId,
    address consumer
  )
    external;

  function removeConsumer(
    uint64 subId,
    address consumer
  )
    external;

  function defundSubscription(
    uint64 subId,
    address to,
    uint96 amount
  )
    external;

  function cancelSubscription(
    uint64 subId,
    address to
  )
    external;
}