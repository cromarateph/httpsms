package com.httpsms

import android.telephony.SmsManager
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class SentReceiverTest {
    @Test
    fun retryableRadioResultIsNotTerminal() {
        assertNull(SentReceiver.terminalFailureReason(SmsManager.RESULT_RIL_SMS_SEND_FAIL_RETRY))
        assertEquals("GENERIC_FAILURE", SentReceiver.terminalFailureReason(SmsManager.RESULT_ERROR_GENERIC_FAILURE))
        assertEquals("UNKNOWN:999", SentReceiver.terminalFailureReason(999))
    }
}
