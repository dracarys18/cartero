package app.cartero.ui.platform

import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import app.cartero.platform.topViewController
import kotlinx.cinterop.ExperimentalForeignApi
import platform.AVFoundation.AVAuthorizationStatusAuthorized
import platform.AVFoundation.AVAuthorizationStatusNotDetermined
import platform.AVFoundation.AVCaptureConnection
import platform.AVFoundation.AVCaptureDevice
import platform.AVFoundation.AVCaptureDeviceInput
import platform.AVFoundation.AVCaptureMetadataOutput
import platform.AVFoundation.AVCaptureMetadataOutputObjectsDelegateProtocol
import platform.AVFoundation.AVCaptureOutput
import platform.AVFoundation.AVCaptureSession
import platform.AVFoundation.AVCaptureVideoPreviewLayer
import platform.AVFoundation.AVLayerVideoGravityResizeAspectFill
import platform.AVFoundation.AVMediaTypeVideo
import platform.AVFoundation.AVMetadataMachineReadableCodeObject
import platform.AVFoundation.AVMetadataObjectTypeQRCode
import platform.AVFoundation.authorizationStatusForMediaType
import platform.AVFoundation.requestAccessForMediaType
import platform.Foundation.NSURL
import platform.UIKit.UIAlertAction
import platform.UIKit.UIAlertActionStyleCancel
import platform.UIKit.UIAlertActionStyleDefault
import platform.UIKit.UIAlertController
import platform.UIKit.UIAlertControllerStyleAlert
import platform.UIKit.UIApplication
import platform.UIKit.UIApplicationOpenSettingsURLString
import platform.UIKit.UIColor
import platform.UIKit.UIInterfaceOrientationLandscapeLeft
import platform.UIKit.UIInterfaceOrientationLandscapeRight
import platform.UIKit.UIInterfaceOrientationPortraitUpsideDown
import platform.UIKit.UIViewController
import platform.darwin.DISPATCH_QUEUE_PRIORITY_DEFAULT
import platform.darwin.dispatch_async
import platform.darwin.dispatch_get_global_queue
import platform.darwin.dispatch_get_main_queue

@Composable
actual fun rememberQrScanner(onScanned: (String) -> Unit): () -> Unit {
    val callback by rememberUpdatedState(onScanned)
    return remember {
        {
            when (AVCaptureDevice.authorizationStatusForMediaType(AVMediaTypeVideo)) {
                AVAuthorizationStatusAuthorized -> presentScanner { callback(it) }
                AVAuthorizationStatusNotDetermined -> AVCaptureDevice.requestAccessForMediaType(AVMediaTypeVideo) { granted ->
                    if (granted) dispatch_async(dispatch_get_main_queue()) { presentScanner { callback(it) } }
                }
                else -> presentCameraSettingsPrompt()
            }
        }
    }
}

private fun presentScanner(onScanned: (String) -> Unit) {
    topViewController()?.presentViewController(QrScannerController(onScanned), animated = true, completion = null)
}

private fun presentCameraSettingsPrompt() {
    val alert = UIAlertController.alertControllerWithTitle(
        title = "Camera access is off",
        message = "Allow camera access for Cartero in Settings to scan link codes.",
        preferredStyle = UIAlertControllerStyleAlert,
    )
    alert.addAction(UIAlertAction.actionWithTitle("Open Settings", UIAlertActionStyleDefault) { _ ->
        NSURL.URLWithString(UIApplicationOpenSettingsURLString)?.let {
            UIApplication.sharedApplication.openURL(it, options = emptyMap<Any?, Any>(), completionHandler = null)
        }
    })
    alert.addAction(UIAlertAction.actionWithTitle("Cancel", UIAlertActionStyleCancel, handler = null))
    topViewController()?.presentViewController(alert, animated = true, completion = null)
}

@OptIn(ExperimentalForeignApi::class)
private class QrScannerController(
    private val onScanned: (String) -> Unit,
) : UIViewController(nibName = null, bundle = null), AVCaptureMetadataOutputObjectsDelegateProtocol {
    private val session = AVCaptureSession()
    private val preview = AVCaptureVideoPreviewLayer(session = session)
    private var scanned = false

    override fun viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = UIColor.blackColor
        val device = AVCaptureDevice.defaultDeviceWithMediaType(AVMediaTypeVideo) ?: return
        val input = AVCaptureDeviceInput.deviceInputWithDevice(device, error = null) ?: return
        val output = AVCaptureMetadataOutput()
        if (!session.canAddInput(input) || !session.canAddOutput(output)) return
        session.addInput(input)
        session.addOutput(output)
        output.setMetadataObjectsDelegate(this, queue = dispatch_get_main_queue())
        output.metadataObjectTypes = listOf(AVMetadataObjectTypeQRCode)
        preview.videoGravity = AVLayerVideoGravityResizeAspectFill
        view.layer.addSublayer(preview)
        dispatch_async(dispatch_get_global_queue(DISPATCH_QUEUE_PRIORITY_DEFAULT.toLong(), 0u)) { session.startRunning() }
    }

    override fun viewDidLayoutSubviews() {
        super.viewDidLayoutSubviews()
        preview.frame = view.layer.bounds
        val angle = when (view.window?.windowScene?.interfaceOrientation) {
            UIInterfaceOrientationLandscapeLeft -> 180.0
            UIInterfaceOrientationLandscapeRight -> 0.0
            UIInterfaceOrientationPortraitUpsideDown -> 270.0
            else -> 90.0
        }
        preview.connection?.takeIf { it.isVideoRotationAngleSupported(angle) }?.videoRotationAngle = angle
    }

    override fun viewWillDisappear(animated: Boolean) {
        super.viewWillDisappear(animated)
        dispatch_async(dispatch_get_global_queue(DISPATCH_QUEUE_PRIORITY_DEFAULT.toLong(), 0u)) { session.stopRunning() }
    }

    override fun captureOutput(
        output: AVCaptureOutput,
        didOutputMetadataObjects: List<*>,
        fromConnection: AVCaptureConnection,
    ) {
        if (scanned) return
        val code = didOutputMetadataObjects.firstNotNullOfOrNull { (it as? AVMetadataMachineReadableCodeObject)?.stringValue } ?: return
        scanned = true
        dismissViewControllerAnimated(true) { onScanned(code) }
    }
}
