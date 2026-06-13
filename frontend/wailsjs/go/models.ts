export namespace main {
	
	export class AdbInfo {
	    available: boolean;
	    path: string;
	    source: string;
	    version: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new AdbInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.path = source["path"];
	        this.source = source["source"];
	        this.version = source["version"];
	        this.message = source["message"];
	    }
	}
	export class Device {
	    serial: string;
	    state: string;
	    model: string;
	    product: string;
	    device: string;
	    transportId: string;
	
	    static createFrom(source: any = {}) {
	        return new Device(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serial = source["serial"];
	        this.state = source["state"];
	        this.model = source["model"];
	        this.product = source["product"];
	        this.device = source["device"];
	        this.transportId = source["transportId"];
	    }
	}
	export class InstallRequest {
	    deviceSerial: string;
	    apkPath: string;
	
	    static createFrom(source: any = {}) {
	        return new InstallRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.deviceSerial = source["deviceSerial"];
	        this.apkPath = source["apkPath"];
	    }
	}
	export class InstallResult {
	    success: boolean;
	    cancelled: boolean;
	    output: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new InstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.cancelled = source["cancelled"];
	        this.output = source["output"];
	        this.error = source["error"];
	    }
	}

}

