export default class CustomContextPad {
  static readonly $inject = [
    'contextPad',
    'create',
    'elementFactory',
    'injector',
    'translate',
    'connect',
    'popupMenu'
  ];

  private readonly create: any;
  private readonly elementFactory: any;
  private readonly injector: any;
  private readonly translate: any;
  private readonly connect: any;
  private readonly popupMenu: any;

  constructor(
    contextPad: any,
    create: any,
    elementFactory: any,
    injector: any,
    translate: any,
    connect: any,
    popupMenu: any
  ) {
    this.create = create;
    this.elementFactory = elementFactory;
    this.injector = injector;
    this.translate = translate;
    this.connect = connect;
    this.popupMenu = popupMenu;

    contextPad.registerProvider(this);
  }

  getContextPadEntries(element: any) {
    const {
      create,
      elementFactory,
      translate
    } = this;

    const actions: any = {};

    if (element.type === 'label') {
      return actions;
    }

    const {
      connect,
      popupMenu
    } = this;

    function appendAction(type: string, className: string, title: string, options?: any) {
      function appendListener(event: any, element: any) {
        const shape = elementFactory.createShape({ type, ...options });
        create.start(event, shape, element);
      }

      return {
        group: 'model',
        className,
        title: translate(title),
        action: {
          dragstart: appendListener,
          click: appendListener
        }
      };
    }

    actions['connect'] = {
      group: 'connect',
      className: 'bpmn-icon-connection-multi',
      title: translate('Connect using sequence flow'),
      action: {
        click: (event: any, element: any) => {
          connect.start(event, element);
        },
        dragstart: (event: any, element: any) => {
          connect.start(event, element);
        }
      }
    };

    actions['replace'] = {
      group: 'edit',
      className: 'bpmn-icon-screw-wrench',
      title: translate('Change type'),
      action: {
        click: (event: any, element: any) => {
          popupMenu.open(element, 'bpmn-replace', event.position);
        }
      }
    };

    if (element.type !== 'bpmn:EndEvent' && element.type !== 'bpmn:Participant' && element.type !== 'bpmn:Lane') {
      actions['append.end-event'] = appendAction(
        'bpmn:EndEvent', 'bpmn-icon-end-event-none', 'Append EndEvent'
      );
      actions['append.exclusive-gateway'] = appendAction(
        'bpmn:ExclusiveGateway', 'bpmn-icon-gateway-xor', 'Append Exclusive Gateway'
      );
      actions['append.parallel-gateway'] = appendAction(
        'bpmn:ParallelGateway', 'bpmn-icon-gateway-parallel', 'Append Parallel Gateway'
      );
      actions['append.task'] = appendAction(
        'bpmn:UserTask', 'bpmn-icon-user-task', 'Append Stage (User Task)'
      );
    }

    actions['delete'] = {
      group: 'edit',
      className: 'bpmn-icon-trash',
      title: translate('Remove'),
      action: {
        click: (_event: any, element: any) => {
          this.injector.get('modeling').removeElements([element]);
        }
      }
    };

    return actions;
  }
}
