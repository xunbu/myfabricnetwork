'use strict';

const { WorkloadModuleBase } = require('@hyperledger/caliper-core');

class WriteWorkload extends WorkloadModuleBase {
    constructor() {
        super();
    }

    async initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext) {
        await super.initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext);
        
        this.prefix = `client_${workerIndex}_`;
        
        // ---------------------------------------------------------
        // ✅ 关键修复：显式初始化计数器为 0
        // ---------------------------------------------------------
        this.txIndex = 0; 
    }

    async submitTransaction() {
        // 现在这里将生成正确的 Key: client_0_0, client_0_1 ...
        const key = this.prefix + this.txIndex;
        const value = `value_content_${this.txIndex}`;

        const myArgs = {
            contractId: 'basic',
            contractFunction: 'PutValue',
            contractArguments: [key, value],
            readOnly: false
        };

        // 计数器递增
        this.txIndex++;
        
        await this.sutAdapter.sendRequests(myArgs);
    }
}

function createWorkloadModule() {
    return new WriteWorkload();
}

module.exports.createWorkloadModule = createWorkloadModule;