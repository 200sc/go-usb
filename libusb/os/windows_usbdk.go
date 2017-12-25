package os

type TransferResult uint8
const(
	TransferFailure TransferResult = iota
	TransferSuccess TransferResult
	TransferSuccessAsync TransferResult
)

type USB_DK_DEVICE_SPEED
const(
	NoSpeed USB_DK_DEVICE_SPEED = iota
	LowSpeed USB_DK_DEVICE_SPEED
	FullSpeed USB_DK_DEVICE_SPEED
	HighSpeed USB_DK_DEVICE_SPEED
	SuperSpee USB_DK_DEVICE_SPEED
)

type USB_DK_TRANSFER_TYPE
const(
	ControlTransferType USB_DK_TRANSFER_TYPE = iota
	BulkTransferType USB_DK_TRANSFER_TYPE
	IntertuptTransferType USB_DK_TRANSFER_TYPE
	IsochronousTransferType USB_DK_TRANSFER_TYPE
)