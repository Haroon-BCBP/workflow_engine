export default class CustomPalette {
  static readonly $inject = [
    'palette',
    'create',
    'elementFactory',
    'spaceTool',
    'lassoTool',
    'handTool',
    'globalConnect',
    'translate'
  ];

  private readonly create: any;
  private readonly elementFactory: any;
  private readonly spaceTool: any;
  private readonly lassoTool: any;
  private readonly handTool: any;
  private readonly globalConnect: any;
  private readonly translate: any;

  constructor(
    palette: any,
    create: any,
    elementFactory: any,
    spaceTool: any,
    lassoTool: any,
    handTool: any,
    globalConnect: any,
    translate: any
  ) {
    this.create = create;
    this.elementFactory = elementFactory;
    this.spaceTool = spaceTool;
    this.lassoTool = lassoTool;
    this.handTool = handTool;
    this.globalConnect = globalConnect;
    this.translate = translate;

    palette.registerProvider(this);
  }

  getPaletteEntries() {
    const {
      create,
      elementFactory,
      spaceTool,
      lassoTool,
      handTool,
      globalConnect,
      translate
    } = this;

    function createAction(type: string, group: string, className: string, title: string, options?: any) {
      function createListener(event: any) {
        const shape = elementFactory.createShape({ type, ...options });
        create.start(event, shape);
      }

      return {
        group,
        className,
        title: translate(title),
        action: {
          dragstart: createListener,
          click: createListener
        }
      };
    }

    return {
      'hand-tool': {
        group: 'tools',
        className: 'bpmn-icon-hand-tool',
        title: translate('Activate the hand tool'),
        action: {
          click: function(event: any) {
            handTool.activateHandTool(event);
          }
        }
      },
      'lasso-tool': {
        group: 'tools',
        className: 'bpmn-icon-lasso-tool',
        title: translate('Activate the lasso tool'),
        action: {
          click: function(event: any) {
            lassoTool.activateSelection(event);
          }
        }
      },
      'space-tool': {
        group: 'tools',
        className: 'bpmn-icon-space-tool',
        title: translate('Activate the create/remove space tool'),
        action: {
          click: function(event: any) {
            spaceTool.activateSelection(event);
          }
        }
      },
      'global-connect-tool': {
        group: 'tools',
        className: 'bpmn-icon-connection-multi',
        title: translate('Activate the global connect tool'),
        action: {
          click: function(event: any) {
            globalConnect.toggle(event);
          }
        }
      },
      'tool-separator': {
        group: 'tools',
        separator: true
      },
      'create.start-event': createAction(
        'bpmn:StartEvent', 'event', 'bpmn-icon-start-event-none', 'Create StartEvent'
      ),
      'create.end-event': createAction(
        'bpmn:EndEvent', 'event', 'bpmn-icon-end-event-none', 'Create EndEvent'
      ),
      'create.exclusive-gateway': createAction(
        'bpmn:ExclusiveGateway', 'gateway', 'bpmn-icon-gateway-xor', 'Create Exclusive Gateway'
      ),
      'create.parallel-gateway': createAction(
        'bpmn:ParallelGateway', 'gateway', 'bpmn-icon-gateway-parallel', 'Create Parallel Gateway'
      ),
      'create.task': createAction(
        'bpmn:UserTask', 'activity', 'bpmn-icon-user-task', 'Create Stage (User Task)'
      ),
      'create.participant-expanded': {
        group: 'collaboration',
        className: 'bpmn-icon-participant',
        title: translate('Create Department (Pool/Lane)'),
        action: {
          dragstart: (_event: any) => {
            const shape = elementFactory.createParticipantShape(true);
            create.start(_event, shape);
          },
          click: (_event: any) => {
            const shape = elementFactory.createParticipantShape(true);
            create.start(_event, shape);
          }
        }
      }
    };
  }
}
