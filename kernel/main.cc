#include <cstring>

#include "WProgram.h"

#include "pb_encode.h"
#include "pb_decode.h"

#include "summer.pb.h"


const size_t input_buffer_length = 1000;
uint8_t input_buffer[input_buffer_length];
size_t input_buffer_bytes = 0;

const size_t output_buffer_length = 1000;
uint8_t output_buffer[output_buffer_length];

bool consumeMessage(const pb_field_t fields[], void *dest_struct) {
	uint64_t length;
	pb_istream_t istream;
	while (true) {
		istream = pb_istream_from_buffer(input_buffer, input_buffer_bytes);
		if (pb_decode_varint(&istream, &length) && ((uint8_t*)istream.state - input_buffer) + length <= input_buffer_bytes) {
			// Got the entire response message in the buffer.
			break;
		}
		if (input_buffer_bytes == input_buffer_length) {
			// Buffer overflow.
			return false;
		}
		// Read from serial port.
		input_buffer_bytes += Serial.readBytes((char*)(input_buffer + input_buffer_bytes), input_buffer_length - input_buffer_bytes);
	}
	// Enough data read and we can parse it.
	istream.bytes_left = length;
	if (!pb_decode(&istream, fields, dest_struct)) {
		// Unmarshalling failure.
		return false;
	}
	// Move unparsed bytes to the beginning of the buffer.
	input_buffer_bytes -= (uint8_t*)istream.state - input_buffer;
	memmove(input_buffer, istream.state, input_buffer_bytes);
	return true;
}

int main() {
	Serial.begin(9600);

	pb_ostream_t sstream;
	sstream.callback = [](pb_ostream_t* stream, const uint8_t* buf, size_t count) -> bool {
		usb_serial_class *usb_serial = (usb_serial_class*)stream->state;
		// Unlike Arduino libraries where write returns amount of bytes written,
		// Teensyduino returns 0 if write succeeds and -1 otherwise.
		return usb_serial->write(buf, count) == 0;
	};
	sstream.state = &Serial;
	sstream.max_size = SIZE_MAX;
	sstream.bytes_written = 0;

	proto_Request request;
	while (consumeMessage(proto_Request_fields, &request)) {
		proto_Response response;
		response.has_sum = true;
		response.sum = request.one + request.two;
		pb_ostream_t ostream = pb_ostream_from_buffer(output_buffer, output_buffer_length);
		if (!pb_encode(&ostream, proto_Response_fields, &response)) {
			// Failed to encode the response.
			return 1;
		}

		if (!pb_encode_varint(&sstream, ostream.bytes_written)) {
			// Failed to write message length to serial port.
			return 1;
		}
		if (Serial.write(output_buffer, ostream.bytes_written) != 0) {
			// Failed to write message to the serial port.
			return 1;
		}
	}
	return 0;
}
