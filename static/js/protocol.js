/**
 * WebOS Protocol Client
 * Implements the binary protocol for browser-server communication
 * Compatible with the Go protocol package
 */
class ProtocolClient {
    /**
     * Protocol opcodes matching Go implementation
     */
    static get Opcodes() {
        return {
            DISPLAY: 1,
            INPUT: 2,
            FILESYSTEM: 3,
            NETWORK: 4,
            PROCESS: 5,
            AUTH: 6,
            CONNECT: 7,
            DISCONNECT: 8,
            PING: 9,
            PONG: 10,
            ERROR: 11
        };
    }

    /**
     * Protocol constants
     */
    static get Constants() {
        return {
            MAGIC_BYTES: new Uint8Array([0x57, 0x45, 0x42, 0x53]), // "WEBS"
            PROTOCOL_VERSION: 1,
            HEADER_SIZE: 18,
            MAX_PAYLOAD_SIZE: 16 * 1024 * 1024 // 16 MB
        };
    }

    /**
     * Get opcode name for debugging
     * @param {number} opcode - Opcode value
     * @returns {string} Opcode name
     */
    static getOpcodeName(opcode) {
        const names = {
            1: 'DISPLAY',
            2: 'INPUT',
            3: 'FILESYSTEM',
            4: 'NETWORK',
            5: 'PROCESS',
            6: 'AUTH',
            7: 'CONNECT',
            8: 'DISCONNECT',
            9: 'PING',
            10: 'PONG',
            11: 'ERROR'
        };
        return names[opcode] || 'UNKNOWN';
    }

    /**
     * Create a message buffer
     * @param {number} opcode - Message opcode
     * @param {Uint8Array} payload - Message payload
     * @returns {Uint8Array} Encoded message
     */
    static encodeMessage(opcode, payload) {
        const payloadLength = payload ? payload.length : 0;

        if (payloadLength > ProtocolClient.Constants.MAX_PAYLOAD_SIZE) {
            throw new Error('Payload exceeds maximum size');
        }

        const totalSize = ProtocolClient.Constants.HEADER_SIZE + payloadLength;
        const buffer = new Uint8Array(totalSize);

        // Write magic bytes at offset 0
        buffer.set(ProtocolClient.Constants.MAGIC_BYTES, 0);

        // Write version at offset 4
        buffer[4] = ProtocolClient.Constants.PROTOCOL_VERSION;

        // Write opcode at offset 5
        buffer[5] = opcode;

        // Write timestamp (8 bytes, big-endian) at offset 6
        const timestamp = BigInt(Date.now()) * 1000000n; // nanoseconds
        const timestampView = new DataView(new ArrayBuffer(8));
        timestampView.setBigUint64(0, timestamp, false);
        buffer.set(new Uint8Array(timestampView.buffer), 6);

        // Write payload length (4 bytes, big-endian) at offset 14
        const lengthView = new DataView(new ArrayBuffer(4));
        lengthView.setUint32(0, payloadLength, false);
        buffer.set(new Uint8Array(lengthView.buffer), 14);

        // Write payload at offset 18
        if (payload && payloadLength > 0) {
            buffer.set(payload, ProtocolClient.Constants.HEADER_SIZE);
        }

        return buffer;
    }

    /**
     * Decode a message from buffer
     * @param {Uint8Array} buffer - Message buffer
     * @returns {Object} Decoded message {opcode, timestamp, payload}
     */
    static decodeMessage(buffer) {
        if (buffer.length < ProtocolClient.Constants.HEADER_SIZE) {
            throw new Error('Buffer too small');
        }

        // Validate magic bytes
        const magic = buffer.slice(0, 4);
        for (let i = 0; i < 4; i++) {
            if (magic[i] !== ProtocolClient.Constants.MAGIC_BYTES[i]) {
                throw new Error('Invalid magic bytes');
            }
        }

        // Read version
        const version = buffer[4];
        if (version !== ProtocolClient.Constants.PROTOCOL_VERSION) {
            throw new Error(`Invalid protocol version: ${version}`);
        }

        // Read opcode
        const opcode = buffer[5];

        // Read timestamp
        const timestampView = new DataView(buffer.buffer, 6, 8);
        const timestamp = timestampView.getBigUint64(0, false);

        // Read payload length
        const lengthView = new DataView(buffer.buffer, 14, 4);
        const payloadLength = lengthView.getUint32(0, false);

        // Validate payload length
        if (payloadLength > ProtocolClient.Constants.MAX_PAYLOAD_SIZE) {
            throw new Error('Payload exceeds maximum size');
        }

        // Validate buffer size
        if (buffer.length - ProtocolClient.Constants.HEADER_SIZE < payloadLength) {
            throw new Error('Buffer too small for payload');
        }

        // Read payload
        let payload = null;
        if (payloadLength > 0) {
            payload = buffer.slice(
                ProtocolClient.Constants.HEADER_SIZE,
                ProtocolClient.Constants.HEADER_SIZE + payloadLength
            );
        }

        return {
            opcode,
            timestamp: Number(timestamp),
            payload
        };
    }

    /**
     * Create a codec for reading/writing binary data
     * @param {Uint8Array} buffer - Buffer to use
     * @returns {Object} Codec interface
     */
    static createCodec(buffer) {
        let pos = 0;

        return {
            /**
             * Read a single byte
             * @returns {number} Byte value
             */
            readByte() {
                if (pos >= buffer.length) throw new Error('End of buffer');
                return buffer[pos++];
            },

            /**
             * Write a single byte
             * @param {number} value - Byte value
             */
            writeByte(value) {
                if (pos >= buffer.length) throw new Error('End of buffer');
                buffer[pos++] = value;
            },

            /**
             * Read a 16-bit big-endian unsigned integer
             * @returns {number} 16-bit value
             */
            readUint16() {
                const view = new DataView(buffer.buffer, pos, 2);
                pos += 2;
                return view.getUint16(0, false);
            },

            /**
             * Write a 16-bit big-endian unsigned integer
             * @param {number} value - 16-bit value
             */
            writeUint16(value) {
                if (pos + 2 > buffer.length) throw new Error('End of buffer');
                const view = new DataView(buffer.buffer, pos, 2);
                view.setUint16(0, value, false);
                pos += 2;
            },

            /**
             * Read a 32-bit big-endian unsigned integer
             * @returns {number} 32-bit value
             */
            readUint32() {
                const view = new DataView(buffer.buffer, pos, 4);
                pos += 4;
                return view.getUint32(0, false);
            },

            /**
             * Write a 32-bit big-endian unsigned integer
             * @param {number} value - 32-bit value
             */
            writeUint32(value) {
                if (pos + 4 > buffer.length) throw new Error('End of buffer');
                const view = new DataView(buffer.buffer, pos, 4);
                view.setUint32(0, value, false);
                pos += 4;
            },

            /**
             * Read a 64-bit big-endian unsigned integer
             * @returns {BigInt} 64-bit value
             */
            readUint64() {
                const view = new DataView(buffer.buffer, pos, 8);
                pos += 8;
                return view.getBigUint64(0, false);
            },

            /**
             * Write a 64-bit big-endian unsigned integer
             * @param {BigInt} value - 64-bit value
             */
            writeUint64(value) {
                if (pos + 8 > buffer.length) throw new Error('End of buffer');
                const view = new DataView(buffer.buffer, pos, 8);
                view.setBigUint64(0, value, false);
                pos += 8;
            },

            /**
             * Read bytes
             * @param {number} n - Number of bytes to read
             * @returns {Uint8Array} Bytes read
             */
            readBytes(n) {
                if (pos + n > buffer.length) throw new Error('End of buffer');
                const result = buffer.slice(pos, pos + n);
                pos += n;
                return result;
            },

            /**
             * Write bytes
             * @param {Uint8Array} bytes - Bytes to write
             */
            writeBytes(bytes) {
                if (pos + bytes.length > buffer.length) throw new Error('End of buffer');
                buffer.set(bytes, pos);
                pos += bytes.length;
            },

            /**
             * Reset position to beginning
             */
            reset() {
                pos = 0;
            },

            /**
             * Get remaining bytes
             * @returns {number} Remaining bytes
             */
            remaining() {
                return buffer.length - pos;
            }
        };
    }
}

// Export for use in different environments
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ProtocolClient;
}

// Also expose as global for browser use
if (typeof window !== 'undefined') {
    window.ProtocolClient = ProtocolClient;
}
