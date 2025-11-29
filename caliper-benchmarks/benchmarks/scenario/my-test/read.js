'use strict';

const { WorkloadModuleBase } = require('@hyperledger/caliper-core');

class ReadWorkload extends WorkloadModuleBase {
    constructor() {
        super();
    }

    async initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext) {
        await super.initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext);
        
        // 1. 设置前缀
        // 必须与 write.js 中的逻辑完全一致，否则拼出来的 key 在数据库里找不到
        this.prefix = `client_${workerIndex}_`;
    }

    async submitTransaction() {
        // 2. 生成要查询的 Key
        // 在 config.yaml 里，write-phase 配置了 txNumber: 20
        // 所以写入的数据 ID 范围是 client_0_0 到 client_0_19
        // 这里我们随机选择一个 ID 进行查询，模拟真实读取请求
        const totalWrittenItems = 20; 
        const randomId = Math.floor(Math.random() * totalWrittenItems);
        
        const key = this.prefix + randomId;

        // 3. 构建请求参数
        const myArgs = {
            contractId: 'basic',            // 对应 test-network.yaml 中的 id
            contractFunction: 'QueryByKey', // 对应 Go 链码中的查询函数名
            contractArguments: [key],       // QueryByKey 需要传入 1 个参数：key
            readOnly: true                  // 查询操作建议开启 readOnly
        };

        // 4. 发送请求
        await this.sutAdapter.sendRequests(myArgs);
    }
}

function createWorkloadModule() {
    return new ReadWorkload();
}

module.exports.createWorkloadModule = createWorkloadModule;